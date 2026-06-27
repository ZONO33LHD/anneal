/**
 * Agent evaluation record (F-027〜F-033). Scores arrive at different times
 * (CI → review → merge → regression), so a record starts `partial` and is
 * promoted to `final` once the merge outcome is known (9.5).
 */
export type ScoreStatus = 'partial' | 'final';

export interface AgentEvaluation {
  run_id: string;
  update_key: string;
  agent_version: string;
  pull_request_number?: number;
  /** Component scores, each 0..100. */
  ci_success_score: number;
  review_burden_score: number;
  risk_prediction_score: number;
  pr_quality_score: number;
  fix_accuracy_score: number;
  merge_outcome_score: number;
  regression_score: number;
  total_score: number;
  score_status: ScoreStatus;
  /** Monotonic update order — stable "recent N" selection (9.5 / T10). */
  seq: number;
  created_at: string;
  updated_at: string;
}

/** Scoring weights for the total (9.5). Regression is tracked separately. */
export const SCORE_WEIGHTS = {
  ci: 0.3,
  review: 0.2,
  risk: 0.15,
  prQuality: 0.15,
  fixAccuracy: 0.15,
  merge: 0.05,
} as const;
