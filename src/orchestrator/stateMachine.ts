import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import { canTransition, type State } from '../domain/states.js';
import { now } from '../util/clock.js';

export class InvalidTransitionError extends Error {
  constructor(
    public readonly from: State,
    public readonly to: State,
  ) {
    super(`Invalid transition: ${from} -> ${to}`);
    this.name = 'InvalidTransitionError';
  }
}

/**
 * Move a record one step along the lifecycle. Returns a NEW record (immutability,
 * per coding-style rules) with the transition appended to its audit history.
 * Throws if the transition is not allowed by the graph (7章).
 */
export function transition(
  record: DependencyUpdate,
  to: State,
  reason: string,
  patch: Partial<DependencyUpdate> = {},
): DependencyUpdate {
  if (record.status === to) {
    // No-op transitions are allowed (idempotent re-delivery) but still merge patch.
    return { ...record, ...patch, updated_at: now() };
  }
  if (!canTransition(record.status, to)) {
    throw new InvalidTransitionError(record.status, to);
  }
  const at = now();
  return {
    ...record,
    ...patch,
    status: to,
    updated_at: at,
    history: [...record.history, { from: record.status, to, reason, at }],
  };
}

/** Mark a record as errored from any state (NF-007 retry-safe). */
export function toError(record: DependencyUpdate, reason: string): DependencyUpdate {
  const at = now();
  return {
    ...record,
    status: 'error',
    updated_at: at,
    history: [...record.history, { from: record.status, to: 'error', reason, at }],
  };
}
