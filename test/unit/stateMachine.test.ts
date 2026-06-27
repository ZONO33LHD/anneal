import { describe, it, expect } from 'vitest';
import {
  transition,
  toError,
  InvalidTransitionError,
} from '../../src/orchestrator/stateMachine.js';
import { canTransition, isTerminal } from '../../src/domain/states.js';
import { makeUpdate } from '../factories.js';

describe('state machine', () => {
  it('allows valid transitions and records history', () => {
    const record = makeUpdate({ status: 'detected' });
    const next = transition(record, 'analyzing', 'start analysis');
    expect(next.status).toBe('analyzing');
    expect(next.history).toHaveLength(1);
    expect(next.history[0]).toMatchObject({ from: 'detected', to: 'analyzing' });
    // immutability
    expect(record.status).toBe('detected');
  });

  it('rejects invalid transitions', () => {
    const record = makeUpdate({ status: 'detected' });
    expect(() => transition(record, 'merged', 'nope')).toThrow(InvalidTransitionError);
  });

  it('treats same-state as an idempotent no-op merge', () => {
    const record = makeUpdate({ status: 'ci_running' });
    const next = transition(record, 'ci_running', 'redelivery', {
      pull_request_number: 5,
    });
    expect(next.status).toBe('ci_running');
    expect(next.history).toHaveLength(0);
    expect(next.pull_request_number).toBe(5);
  });

  it('can move to error from any state', () => {
    const record = makeUpdate({ status: 'pr_creating' });
    const next = toError(record, 'boom');
    expect(next.status).toBe('error');
  });

  it('marks terminal states', () => {
    expect(isTerminal('done')).toBe(true);
    expect(isTerminal('closed')).toBe(true);
    expect(isTerminal('superseded')).toBe(true);
    expect(isTerminal('analyzing')).toBe(false);
  });

  it('exposes the transition table', () => {
    expect(canTransition('merged', 'monitoring_regression')).toBe(true);
    expect(canTransition('done', 'analyzing')).toBe(false);
  });
});
