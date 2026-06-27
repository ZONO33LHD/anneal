import type { AnnealContext } from './context.js';
import { tick } from './runner.js';
import { log } from '../util/logger.js';

/**
 * T9 reconciliation loop. In local mode the store IS the source of truth, so a
 * reconcile is a single tick that re-derives progress for any record left behind
 * by a missed event. In real mode this is where GitHub state would be re-queried
 * and merged back into the records (NF-007, 6.2).
 */
export async function reconcile(ctx: AnnealContext): Promise<number> {
  const changed = await tick(ctx);
  log.info('reconcile complete', { advanced: changed });
  return changed;
}
