import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import { analyzeImpact } from '../agent/impactAnalyzer.js';
import { composePr } from '../agent/prComposer.js';
import { classifyCiFailure } from '../agent/ciFailureClassifier.js';
import { getEcosystem } from '../ecosystems/index.js';
import { transition } from './stateMachine.js';
import type { AnnealContext } from './context.js';
import { log } from '../util/logger.js';

/** detected → analyzing: run impact analysis (F-007〜F-012). */
export async function analyzeStep(
  ctx: AnnealContext,
  record: DependencyUpdate,
): Promise<DependencyUpdate> {
  const repoPath = record.repoPath ?? '.';
  const impact = await analyzeImpact(record, repoPath, ctx.llm);
  log.step('analyzing', { key: record.update_key, risk: impact.riskLevel });
  return transition(record, 'analyzing', 'impact analysis complete', {
    impact,
    risk_level: impact.riskLevel,
  });
}

/** pr_creating → pr_created: apply the bump, compose + open the PR (F-013〜F-018). */
export async function createPrStep(
  ctx: AnnealContext,
  record: DependencyUpdate,
): Promise<DependencyUpdate> {
  const eco = getEcosystem(record.ecosystem);
  let changedFiles: string[] = [];
  if (eco && record.repoPath) {
    changedFiles = await eco.applyUpdate(
      record.repoPath,
      record.package_name,
      record.target_version,
    );
  }
  const pr = await composePr(record, changedFiles, ctx.llm);
  const ref = await ctx.git.createBranchAndPr({
    repository: record.repository,
    base: 'main',
    branch: pr.branch,
    title: pr.title,
    body: pr.body,
    changedFiles,
  });

  await ctx.notifier.notify({
    level: record.cve ? 'priority' : 'info',
    title: `PR opened: ${pr.title}`,
    body: `Risk ${record.impact?.riskLevel ?? 'n/a'} · Confidence ${
      record.impact?.confidence.toFixed(2) ?? 'n/a'
    }`,
    url: ref.url,
  });
  log.step('pr_created', { key: record.update_key, url: ref.url });

  return transition(record, 'pr_created', 'branch + changes + PR created', {
    branch: pr.branch,
    pull_request_url: ref.url,
    pull_request_number: ref.number,
  });
}

/** ci_running → ci_passed | ci_failed (T4). */
export async function ciStep(
  ctx: AnnealContext,
  record: DependencyUpdate,
): Promise<DependencyUpdate> {
  const attempt = record.ci?.attempts ?? 0;
  const shouldFailFirst =
    record.update_type === 'minor' && (record.impact?.usageSites.length ?? 0) > 0;
  const result = await ctx.git.checkCi({
    repository: record.repository,
    prNumber: record.pull_request_number ?? 0,
    branch: record.branch ?? '',
    attempt,
    shouldFailFirst,
  });

  if (result.passed) {
    log.step('ci_passed', { key: record.update_key, attempt });
    return transition(record, 'ci_passed', 'CI succeeded', {
      ci: { status: 'passed', attempts: attempt },
    });
  }
  log.step('ci_failed', { key: record.update_key, attempt });
  return transition(record, 'ci_failed', 'CI failed', {
    ci: { status: 'failed', attempts: attempt, logSummary: result.logSummary },
  });
}

/** ci_failed → fixing | awaiting_review (F-023〜F-026). */
export async function fixStep(
  ctx: AnnealContext,
  record: DependencyUpdate,
): Promise<DependencyUpdate> {
  const logSummary = record.ci?.logSummary ?? 'unknown failure';
  const analysis = await classifyCiFailure(logSummary, ctx.llm);
  const attempts = record.ci?.attempts ?? 0;

  // NF-009: stop self-fixing after repeated failures.
  if (attempts >= 2) {
    log.warn('on_hold: repeated CI failures', { key: record.update_key });
    // ci_failed can't go to on_hold directly; route via awaiting_review.
    return transition(record, 'awaiting_review', 'repeated CI failures → human', {
      ci: { ...record.ci!, failureCategory: analysis.category, fixable: false },
    });
  }

  if (!analysis.fixable) {
    log.step('awaiting_review', {
      key: record.update_key,
      reason: analysis.category,
    });
    return transition(record, 'awaiting_review', `unfixable: ${analysis.category}`, {
      ci: { ...record.ci!, failureCategory: analysis.category, fixable: false },
    });
  }

  await ctx.git.pushFix({
    repository: record.repository,
    branch: record.branch ?? '',
    prNumber: record.pull_request_number ?? 0,
    message: `fix: address ${analysis.category} after dependency bump`,
    changedFiles: ['(auto-fix)'],
  });
  await ctx.notifier.notify({
    level: 'info',
    title: `Anneal auto-fixed CI: ${analysis.category}`,
    body: analysis.summary,
    url: record.pull_request_url,
  });
  log.step('fixing', { key: record.update_key, category: analysis.category });

  return transition(record, 'fixing', `auto-fix for ${analysis.category}`, {
    ci: {
      status: 'running',
      attempts: attempts + 1,
      failureCategory: analysis.category,
      fixable: true,
    },
  });
}
