import { shouldFire, runAnnealing } from '../improvement/annealing.js';
import { dispatch } from './triggers.js';
import type { AnnealContext } from './context.js';
import { log } from '../util/logger.js';

/**
 * One tick: advance every active record one meaningful step. This is the unit
 * the Scheduler (T1) / reconcile loop (T9) invokes — read records, step the
 * state machine, write back.
 */
export async function tick(ctx: AnnealContext): Promise<number> {
  const active = await ctx.store.listActiveUpdates();
  let changed = 0;
  for (const record of active) {
    const result = await dispatch(ctx, record);
    if (result.changed) changed += 1;
  }
  return changed;
}

/**
 * Drive the whole lifecycle to a standstill (used by `demo`): keep ticking until
 * no record changes or the round budget is exhausted (safety bound).
 */
export async function drive(ctx: AnnealContext, maxRounds = 100): Promise<void> {
  for (let round = 0; round < maxRounds; round += 1) {
    const changed = await tick(ctx);
    if (changed === 0) return;
  }
  log.warn('drive: hit max rounds; some records may still be active');
}

/** Fire the Annealing Loop if the rolling score average dropped (T10). */
export async function maybeAnneal(ctx: AnnealContext): Promise<void> {
  const { fire, average, sampled } = await shouldFire(
    ctx.store,
    ctx.settings.improveWindow,
    ctx.settings.lowScoreThreshold,
  );
  if (sampled === 0) return;
  log.info('score check', { average, sampled, threshold: ctx.settings.lowScoreThreshold });
  if (fire) {
    await runAnnealing(ctx.store, ctx.llm, ctx.notifier, average);
  }
}
