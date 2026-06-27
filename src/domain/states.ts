/**
 * Dependency Update Lifecycle — the state machine that is the backbone of Anneal
 * (要件書 7章). Every trigger (8章) does exactly one thing: read a record, move
 * it one step along this graph, and write it back.
 */
export const STATES = [
  'detected',
  'analyzing',
  'awaiting_approval',
  'pr_creating',
  'pr_created',
  'ci_running',
  'ci_passed',
  'ci_failed',
  'fixing',
  'awaiting_review',
  'changes_requested',
  'merged',
  'monitoring_regression',
  'done',
  'regressed',
  'superseded',
  'on_hold',
  'closed',
  'error',
] as const;

export type State = (typeof STATES)[number];

/**
 * Allowed transitions. A move not listed here is rejected by the state machine,
 * which keeps the asynchronous lifecycle from drifting into invalid states.
 */
export const TRANSITIONS: Record<State, State[]> = {
  detected: ['analyzing', 'superseded', 'error'],
  analyzing: ['awaiting_approval', 'pr_creating', 'error'],
  awaiting_approval: ['pr_creating', 'closed'],
  pr_creating: ['pr_created', 'error'],
  pr_created: ['ci_running', 'superseded'],
  ci_running: ['ci_passed', 'ci_failed'],
  ci_passed: ['awaiting_review'],
  ci_failed: ['fixing', 'awaiting_review'],
  fixing: ['ci_running', 'on_hold'],
  awaiting_review: ['changes_requested', 'merged', 'closed'],
  changes_requested: ['fixing'],
  merged: ['monitoring_regression'],
  monitoring_regression: ['done', 'regressed'],
  on_hold: ['awaiting_review'],
  regressed: ['awaiting_review'],
  // Terminal states.
  done: [],
  closed: [],
  superseded: [],
  // error is recoverable: a retry re-enters the pipeline from `detected`.
  error: ['detected', 'analyzing', 'pr_creating', 'closed'],
};

/** Terminal states never need further processing and are not "active". */
export const TERMINAL_STATES: State[] = ['done', 'closed', 'superseded'];

export function isTerminal(state: State): boolean {
  return TERMINAL_STATES.includes(state);
}

/**
 * Active = still in flight. Used by NF-008 duplicate-prevention: a new record is
 * not created for an update_key that already has an active record. `error` counts
 * as active because it is retryable.
 */
export function isActive(state: State): boolean {
  return !isTerminal(state);
}

export function canTransition(from: State, to: State): boolean {
  return TRANSITIONS[from].includes(to);
}
