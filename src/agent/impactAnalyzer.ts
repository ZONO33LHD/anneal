import { readdir, readFile, stat } from 'node:fs/promises';
import { join, relative } from 'node:path';
import type {
  DependencyUpdate,
  ImpactAnalysis,
  RiskLevel,
} from '../domain/dependencyUpdate.js';
import type { Llm } from '../providers/llm/llm.js';

const SKIP_DIRS = new Set(['node_modules', '.git', 'dist', 'vendor', '.anneal']);
const TEXT_EXT = /\.(ts|tsx|js|jsx|mjs|cjs|go|json)$/;
const MAX_FILES = 500; // bound the scan so large repos stay fast

async function walk(dir: string, root: string, acc: string[]): Promise<void> {
  if (acc.length >= MAX_FILES) return;
  let entries;
  try {
    entries = await readdir(dir, { withFileTypes: true });
  } catch {
    return;
  }
  for (const entry of entries) {
    if (acc.length >= MAX_FILES) return;
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      if (SKIP_DIRS.has(entry.name)) continue;
      await walk(full, root, acc);
    } else if (TEXT_EXT.test(entry.name)) {
      acc.push(full);
    }
  }
}

/** Build a search pattern for the package across npm + go import styles. */
function usagePattern(packageName: string): RegExp {
  const escaped = packageName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp(`(from\\s+['"]${escaped}|require\\(['"]${escaped}|['"]${escaped})`);
}

/**
 * Impact analysis. Deterministically locates usage sites and derives a risk
 * level, then asks the LLM for a one-paragraph human summary (enrichment only —
 * the risk decision itself stays rule-based so it is predictable and auditable).
 */
export async function analyzeImpact(
  update: DependencyUpdate,
  repoPath: string,
  llm: Llm,
): Promise<ImpactAnalysis> {
  const files: string[] = [];
  await walk(repoPath, repoPath, files);
  const pattern = usagePattern(update.package_name);

  const usageSites: string[] = [];
  for (const file of files) {
    try {
      const info = await stat(file);
      if (info.size > 512 * 1024) continue;
      const content = await readFile(file, 'utf8');
      if (pattern.test(content)) usageSites.push(relative(repoPath, file));
    } catch {
      // ignore unreadable files
    }
  }

  const hasBreakingChange = update.update_type === 'major';
  const riskLevel = deriveRisk(update.update_type, usageSites.length, hasBreakingChange);
  const confidence = deriveConfidence(update.update_type, usageSites.length);

  const summary = await summarize(update, usageSites.length, riskLevel, llm);

  return {
    usageSites,
    hasBreakingChange,
    affectedFiles: usageSites,
    riskLevel,
    confidence,
    summary,
  };
}

function deriveRisk(
  updateType: string,
  siteCount: number,
  breaking: boolean,
): RiskLevel {
  if (breaking || updateType === 'major') return 'high';
  if (updateType === 'minor' && siteCount > 10) return 'high';
  if (updateType === 'minor' && siteCount > 0) return 'medium';
  if (siteCount > 20) return 'medium';
  return 'low';
}

function deriveConfidence(updateType: string, siteCount: number): number {
  if (updateType === 'patch') return siteCount > 0 ? 0.9 : 0.95;
  if (updateType === 'minor') return siteCount > 10 ? 0.55 : 0.8;
  return 0.4; // major
}

async function summarize(
  update: DependencyUpdate,
  siteCount: number,
  risk: RiskLevel,
  llm: Llm,
): Promise<string> {
  const prompt = `[impact] Summarize the impact of bumping ${update.package_name} ` +
    `from ${update.current_version} to ${update.target_version} ` +
    `(${update.update_type}, ${siteCount} usage sites, risk=${risk}).`;
  const enriched = await llm.generate({ prompt, temperature: 0.2 });
  const base = `${update.package_name} ${update.current_version} → ${update.target_version} ` +
    `(${update.update_type}); ${siteCount} usage site(s); risk ${risk}.`;
  return enriched && enriched !== 'OK' ? `${base} ${enriched}` : base;
}
