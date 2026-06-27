import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import { decide } from '../agent/decisionRules.js';
import { buildEvaluation } from '../scoring/partialScore.js';
import { recordIfLowScore } from '../improvement/failureStore.js';
import { transition, toError } from './stateMachine.js';
import { analyzeStep, createPrStep, ciStep, fixStep } from './pipeline.js';
import type { AnnealContext } from './context.js';
import { log } from '../util/logger.js';

/** States whose entry should (re)compute the evaluation record. */
const SCORE_STATES = new Set([
  'ci_passed',
  'ci_failed',
  'awaiting_review',
  'changes_requested',
  'merged',
  'monitoring_regression',
  'done',
  'regressed',
  'closed',
]);

export interface DispatchResult {
  changed: boolean;
  record: DependencyUpdate;
}

/** Simulated reviewer comment count, scaled by predicted risk (demo only). */
function simulatedComments(record: DependencyUpdate): number {
  switch (record.impact?.riskLevel) {
    case 'high':
      return 3;
    case 'medium':
      return 1;
    default:
      return 0;
  }
}

/**
 * Advance a single record exactly one meaningful step (6.2 single discipline).
 * Persists the new record and (re)computes its evaluation. Human gates and async
 * boundaries stop here in real mode; in simulate mode they auto-advance so the
 * full lifecycle is demonstrable.
 */
export async function dispatch(
  ctx: AnnealContext,
  record: DependencyUpdate,
): Promise<DispatchResult> {
  let next: DependencyUpdate | null = null;
  try {
    next = await advance(ctx, record);
  } catch (err) {
    log.error('dispatch failed; moving to error', {
      key: record.update_key,
      err: String(err),
    });
    next = toError(record, String(err));
  }

  if (!next || next.status === record.status) {
    if (next && next !== record) await ctx.store.putUpdate(next);
    return { changed: false, record: next ?? record };
  }

  await ctx.store.putUpdate(next);
  await maybeScore(ctx, next);
  return { changed: true, record: next };
}

async function advance(
  ctx: AnnealContext,
  record: DependencyUpdate,
): Promise<DependencyUpdate | null> {
  switch (record.status) {
    case 'detected':
      return analyzeStep(ctx, record);

    case 'analyzing': {
      const decision = decide(record);
      if (decision.decision === 'human') {
        await ctx.notifier.notify({
          level: 'approval',
          title: `Human approval required: ${record.package_name}`,
          body: decision.reasons.join('; '),
        });
        return transition(
          record,
          'awaiting_approval',
          `human: ${decision.reasons.join('; ')}`,
        );
      }
      return transition(
        record,
        'pr_creating',
        `auto: ${decision.reasons.join('; ')}`,
      );
    }

    case 'awaiting_approval':
      if (!ctx.simulate) return null; // wait for a human
      return transition(record, 'pr_creating', 'simulated approval');

    case 'pr_creating':
      return createPrStep(ctx, record);

    case 'pr_created':
      return transition(record, 'ci_running', 'CI started', {
        ci: { status: 'running', attempts: 0 },
      });

    case 'ci_running':
      return ciStep(ctx, record);

    case 'ci_passed':
      return transition(record, 'awaiting_review', 'CI passed → review');

    case 'ci_failed':
      return fixStep(ctx, record);

    case 'fixing':
      return transition(record, 'ci_running', 're-run CI after fix');

    case 'changes_requested':
      if (!ctx.simulate) return null;
      return transition(record, 'fixing', 'addressing review feedback');

    case 'awaiting_review':
      if (!ctx.simulate) return null; // wait for review/merge webhook
      if (record.ci?.status === 'failed') {
        // A human would not merge a red PR — reject it.
        return transition(record, 'closed', 'simulated human rejection (CI red)', {
          reviewCommentCount: simulatedComments(record),
        });
      }
      return transition(record, 'merged', 'simulated human merge', {
        reviewCommentCount: simulatedComments(record),
      });

    case 'merged':
      await ctx.notifier.notify({
        level: 'success',
        title: `Merged: ${record.package_name} → ${record.target_version}`,
        body: 'Entering regression monitoring window.',
        url: record.pull_request_url,
      });
      return transition(record, 'monitoring_regression', 'merged');

    case 'monitoring_regression':
      if (!ctx.simulate) return null; // closed by the regression scheduler (T8)
      return transition(record, 'done', 'no regression detected');

    default:
      return null; // terminal / on_hold / error
  }
}

async function maybeScore(
  ctx: AnnealContext,
  record: DependencyUpdate,
): Promise<void> {
  if (!SCORE_STATES.has(record.status)) return;
  const prev = await ctx.store.getEvaluation(record.update_key);
  const evaluation = buildEvaluation(record, prev);
  await ctx.store.putEvaluation(evaluation);
  if (evaluation.score_status === 'final') {
    await recordIfLowScore(
      ctx.store,
      record,
      evaluation,
      ctx.settings.lowScoreThreshold,
    );
  }
}
