import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import type { AgentEvaluation } from '../domain/evaluation.js';
import type { Store } from '../store/store.js';
import { now } from '../util/clock.js';
import { log } from '../util/logger.js';

/**
 * Persist a low-scoring case as a failure example so the Annealing Loop has
 * material to learn from. Only finalized evaluations below threshold count.
 */
export async function recordIfLowScore(
  store: Store,
  update: DependencyUpdate,
  evaluation: AgentEvaluation,
  threshold: number,
): Promise<boolean> {
  if (evaluation.score_status !== 'final') return false;
  if (evaluation.total_score >= threshold) return false;

  // Idempotent: only record one failure case per update_key.
  const existing = await store.listFailures();
  if (existing.some((f) => f.update_key === update.update_key)) return false;

  await store.putFailure({
    update_key: update.update_key,
    agent_version: update.agent_version,
    total_score: evaluation.total_score,
    reason: `total ${evaluation.total_score} < ${threshold}`,
    snapshot: {
      update_type: update.update_type,
      risk_level: update.risk_level,
      ci: update.ci,
      impact: update.impact
        ? { riskLevel: update.impact.riskLevel, sites: update.impact.usageSites.length }
        : undefined,
    },
    created_at: now(),
  });
  log.warn('recorded failure case', {
    update: update.update_key,
    score: evaluation.total_score,
  });
  return true;
}
