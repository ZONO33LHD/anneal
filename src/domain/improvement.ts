/**
 * Self-improvement record (Annealing Loop, F-034〜F-042). In this build we cover
 * up to candidate generation (Should scope); A/B adoption is future work.
 */
export type ImprovementStatus = 'candidate' | 'adopted' | 'rolled_back';

export interface FailureCase {
  update_key: string;
  agent_version: string;
  total_score: number;
  reason: string;
  snapshot: Record<string, unknown>;
  created_at: string;
}

export interface AgentImprovement {
  improvement_id: string;
  trigger: string;
  target: 'prompt' | 'tool' | 'rule' | 'weights';
  previous_version: string;
  candidate_version: string;
  hypothesis: string;
  proposedChange: string;
  status: ImprovementStatus;
  created_at: string;
}
