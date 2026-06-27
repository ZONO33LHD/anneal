import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import type { AgentEvaluation } from '../domain/evaluation.js';
import type { AgentImprovement, FailureCase } from '../domain/improvement.js';

/**
 * Repository-pattern interface. The persistent store is the single source of
 * truth; the orchestrator reads, advances one step, and writes back.
 */
export interface Store {
  // --- DependencyUpdate ---
  getUpdate(updateKey: string): Promise<DependencyUpdate | undefined>;
  /** Active record (non-terminal) for an update_key — basis for duplicate prevention. */
  getActiveUpdate(updateKey: string): Promise<DependencyUpdate | undefined>;
  putUpdate(record: DependencyUpdate): Promise<void>;
  listUpdates(): Promise<DependencyUpdate[]>;
  /** Non-terminal records, used by tick/reconcile loops. */
  listActiveUpdates(): Promise<DependencyUpdate[]>;

  // --- Evaluations ---
  getEvaluation(updateKey: string): Promise<AgentEvaluation | undefined>;
  putEvaluation(record: AgentEvaluation): Promise<void>;
  listEvaluations(): Promise<AgentEvaluation[]>;

  // --- Improvements & failures (Annealing Loop) ---
  putFailure(record: FailureCase): Promise<void>;
  listFailures(): Promise<FailureCase[]>;
  putImprovement(record: AgentImprovement): Promise<void>;
  listImprovements(): Promise<AgentImprovement[]>;
}
