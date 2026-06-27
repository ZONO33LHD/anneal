import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { parse } from 'yaml';
import { log } from '../util/logger.js';

/**
 * Per-repository configuration committed as `.anneal.yml`.
 * Repo settings override the organisation defaults represented here.
 */
export interface RepoConfig {
  baseBranch: string;
  /** Package names (or glob-ish prefixes) to skip entirely. */
  ignore: string[];
  /** Update types eligible for automatic PR creation. */
  autoPrTypes: ('patch' | 'minor' | 'major')[];
  /** Regression monitoring window in days. */
  regressionWindowDays: number;
}

export const DEFAULT_REPO_CONFIG: RepoConfig = {
  baseBranch: 'main',
  ignore: [],
  autoPrTypes: ['patch', 'minor'],
  regressionWindowDays: 7,
};

export async function loadRepoConfig(repoPath: string): Promise<RepoConfig> {
  try {
    const raw = await readFile(join(repoPath, '.anneal.yml'), 'utf8');
    const parsed = (parse(raw) ?? {}) as Partial<RepoConfig>;
    return { ...DEFAULT_REPO_CONFIG, ...parsed };
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code !== 'ENOENT') {
      log.warn('failed to read .anneal.yml; using defaults', { err: String(err) });
    }
    return { ...DEFAULT_REPO_CONFIG };
  }
}

/** True when the package should be ignored per repo config. */
export function isIgnored(cfg: RepoConfig, packageName: string): boolean {
  return cfg.ignore.some(
    (pattern) => packageName === pattern || packageName.startsWith(pattern.replace(/\*$/, '')),
  );
}
