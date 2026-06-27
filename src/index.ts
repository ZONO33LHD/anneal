#!/usr/bin/env node
import { cp, rm, mkdir } from 'node:fs/promises';
import { resolve, join } from 'node:path';
import { Command } from 'commander';
import { loadConfig } from './config/config.js';
import { makeContext } from './orchestrator/context.js';
import { scanRepository } from './orchestrator/scanner.js';
import { tick, drive, maybeAnneal } from './orchestrator/runner.js';
import { reconcile } from './orchestrator/reconcile.js';
import { setVerbose, log } from './util/logger.js';

const program = new Command();

program
  .name('anneal')
  .description('Self-improving dependency & security update agent')
  .option('-v, --verbose', 'verbose logging')
  .hook('preAction', (cmd) => {
    if (cmd.opts().verbose) setVerbose(true);
  });

program
  .command('scan')
  .description('Scan a repository for updates and create detected records')
  .argument('<repoPath>', 'path to the target repository')
  .option('-r, --repository <name>', 'logical repository name (owner/repo)')
  .action(async (repoPath: string, opts: { repository?: string }) => {
    const ctx = makeContext(loadConfig());
    const result = await scanRepository(ctx, repoPath, opts.repository);
    log.info('scan complete', { created: result.created.length, skipped: result.skipped });
  });

program
  .command('tick')
  .description('Advance every active record one step')
  .action(async () => {
    const ctx = makeContext(loadConfig());
    const changed = await tick(ctx);
    await maybeAnneal(ctx);
    log.info('tick complete', { advanced: changed });
  });

program
  .command('reconcile')
  .description('Reconciliation loop: catch up records left behind by missed events')
  .action(async () => {
    const ctx = makeContext(loadConfig());
    await reconcile(ctx);
    await maybeAnneal(ctx);
  });

program
  .command('improve')
  .description('Manually run the Annealing Loop score check')
  .action(async () => {
    const ctx = makeContext(loadConfig());
    await maybeAnneal(ctx);
  });

program
  .command('status')
  .description('Print a summary of records and scores')
  .action(async () => {
    const ctx = makeContext(loadConfig());
    const updates = await ctx.store.listUpdates();
    const evals = await ctx.store.listEvaluations();
    const improvements = await ctx.store.listImprovements();
    log.info('status', {
      updates: updates.length,
      evaluations: evals.length,
      improvements: improvements.length,
    });
    for (const u of updates) {
      const ev = evals.find((e) => e.update_key === u.update_key);
      console.log(
        `  ${u.status.padEnd(22)} ${u.package_name}@${u.target_version}` +
          (ev ? ` — ${ev.score_status} ${ev.total_score}` : ''),
      );
    }
  });

program
  .command('demo')
  .description('Run the full lifecycle end-to-end on the bundled fixture (all-mock)')
  .action(async () => {
    await runDemo();
  });

program.parseAsync(process.argv).catch((err) => {
  log.error('fatal', { err: String(err) });
  process.exit(1);
});

/**
 * Demo: copies the fixture to an isolated working dir, runs the whole mock
 * pipeline, and shows detection → PR → CI self-heal → scoring → Annealing Loop.
 * Uses a raised low-score threshold so the loop reliably fires.
 */
async function runDemo(): Promise<void> {
  setVerbose(true);
  const work = resolve('.anneal/work/sample-repo');
  const storePath = resolve('.anneal/demo-store.json');
  await rm(work, { recursive: true, force: true });
  await rm(storePath, { force: true });
  await mkdir(join(work, '..'), { recursive: true });
  await cp(resolve('fixtures/sample-repo'), work, { recursive: true });

  const ctx = makeContext(loadConfig({ forceMock: true, storePath }));
  ctx.settings.lowScoreThreshold = 90; // ensure the Annealing Loop has material

  console.log('\n=== 1) Detect ===');
  const scan = await scanRepository(ctx, work, 'acme/sample-repo');
  log.info('detected', { candidates: scan.created.length });

  console.log('\n=== 2) Drive lifecycle (analyze → PR → CI → self-heal → merge) ===');
  await drive(ctx);

  console.log('\n=== 3) Scores ===');
  const evals = await ctx.store.listEvaluations();
  for (const e of evals.sort((a, b) => b.total_score - a.total_score)) {
    console.log(
      `  ${e.update_key.split('::')[1]?.padEnd(28)} ${e.score_status} total=${e.total_score}`,
    );
  }

  console.log('\n=== 4) 🔥 Annealing Loop ===');
  await maybeAnneal(ctx);

  console.log('\n=== Done. Store: ' + storePath + ' ===\n');
}
