import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import type { AgentEvaluation, ScoreStatus } from '../domain/evaluation.js';
import { now } from '../util/clock.js';
import { genId, nextSeq } from '../util/ids.js';
import { computeComponents, computeTotal } from './scorer.js';

/** Statuses at which the score is considered final (9.5). */
const FINAL_STATUSES = ['merged', 'monitoring_regression', 'done', 'regressed', 'closed'];

function scoreStatus(u: DependencyUpdate): ScoreStatus {
  return FINAL_STATUSES.includes(u.status) ? 'final' : 'partial';
}

/**
 * Build or update an evaluation for a record. Scores arrive over time (CI →
 * review → merge → regression); this re-computes components from the current
 * record state and promotes `partial` → `final` once the outcome is known.
 */
export function buildEvaluation(
  u: DependencyUpdate,
  previous?: AgentEvaluation,
): AgentEvaluation {
  const c = computeComponents(u);
  const total = computeTotal(c);
  return {
    run_id: previous?.run_id ?? genId('run'),
    update_key: u.update_key,
    agent_version: u.agent_version,
    pull_request_number: u.pull_request_number,
    ci_success_score: c.ci,
    review_burden_score: c.review,
    risk_prediction_score: c.risk,
    pr_quality_score: c.prQuality,
    fix_accuracy_score: c.fixAccuracy,
    merge_outcome_score: c.merge,
    regression_score: c.regression,
    total_score: total,
    score_status: scoreStatus(u),
    seq: nextSeq(),
    created_at: previous?.created_at ?? now(),
    updated_at: now(),
  };
}
