import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { cp, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { makeTestContext, CapturingNotifier } from '../testContext.js';
import { scanRepository } from '../../src/orchestrator/scanner.js';
import { drive, maybeAnneal } from '../../src/orchestrator/runner.js';
import type { AnnealContext } from '../../src/orchestrator/context.js';

const fixture = join(process.cwd(), 'fixtures', 'sample-repo');
let work: string;
let ctx: AnnealContext;

beforeEach(async () => {
  work = join(tmpdir(), `anneal-e2e-${process.pid}-${Math.floor(performance.now())}`);
  await cp(fixture, work, { recursive: true });
  ctx = makeTestContext({ notifier: new CapturingNotifier() });
});

afterEach(async () => {
  await rm(work, { recursive: true, force: true });
});

describe('full lifecycle (mock)', () => {
  it('detects candidates from npm and go manifests', async () => {
    const result = await scanRepository(ctx, work, 'acme/sample-repo');
    const names = result.created.map((u) => u.package_name);
    expect(names).toContain('lodash');
    expect(names).toContain('chalk');
    expect(names).toContain('golang.org/x/crypto');
    expect(result.created.length).toBeGreaterThanOrEqual(6);
  });

  it('is idempotent: re-scanning creates no duplicate active records (NF-008)', async () => {
    await scanRepository(ctx, work, 'acme/sample-repo');
    const second = await scanRepository(ctx, work, 'acme/sample-repo');
    expect(second.created).toHaveLength(0);
    // active set unchanged
    const beforeCount = (await ctx.store.listUpdates()).length;
    await scanRepository(ctx, work, 'acme/sample-repo');
    expect((await ctx.store.listUpdates()).length).toBe(beforeCount);
  });

  it('drives the security patch to done with a high score', async () => {
    await scanRepository(ctx, work, 'acme/sample-repo');
    await drive(ctx);
    const updates = await ctx.store.listUpdates();
    const lodash = updates.find((u) => u.package_name === 'lodash')!;
    expect(lodash.status).toBe('done');
    const evals = await ctx.store.listEvaluations();
    const lodashEval = evals.find((e) => e.update_key === lodash.update_key)!;
    expect(lodashEval.score_status).toBe('final');
    expect(lodashEval.total_score).toBeGreaterThan(90);
  });

  it('routes the major bump through human approval', async () => {
    await scanRepository(ctx, work, 'acme/sample-repo');
    await drive(ctx);
    const chalk = (await ctx.store.listUpdates()).find((u) => u.package_name === 'chalk')!;
    const sawApproval = chalk.history.some((h) => h.to === 'awaiting_approval');
    expect(sawApproval).toBe(true);
  });

  it('self-heals a failing minor CI run', async () => {
    await scanRepository(ctx, work, 'acme/sample-repo');
    await drive(ctx);
    const axios = (await ctx.store.listUpdates()).find((u) => u.package_name === 'axios')!;
    const failedThenFixed =
      axios.history.some((h) => h.to === 'ci_failed') &&
      axios.history.some((h) => h.to === 'fixing') &&
      axios.history.some((h) => h.to === 'ci_passed');
    expect(failedThenFixed).toBe(true);
  });

  it('fires the Annealing Loop when scores dip', async () => {
    await scanRepository(ctx, work, 'acme/sample-repo');
    await drive(ctx);
    await maybeAnneal(ctx);
    const improvements = await ctx.store.listImprovements();
    expect(improvements.length).toBeGreaterThanOrEqual(1);
    expect(improvements[0]!.status).toBe('candidate');
  });
});
