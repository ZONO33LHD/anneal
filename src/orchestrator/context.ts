import type { AnnealConfig } from '../config/config.js';

/**
 * Runtime context passed to orchestrator handlers. `simulate` is true when no
 * real git backend is configured (mock) — in that mode the orchestrator auto-
 * advances human gates and CI so the full lifecycle is demonstrable locally.
 * In real mode, human gates and CI/merge are driven by webhooks/reconcile.
 */
export interface AnnealContext extends AnnealConfig {
  simulate: boolean;
}

export function makeContext(config: AnnealConfig): AnnealContext {
  return { ...config, simulate: config.git.name === 'mock' };
}
