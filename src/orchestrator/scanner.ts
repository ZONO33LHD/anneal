import { resolve } from 'node:path';
import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import { updateKey } from '../domain/dependencyUpdate.js';
import { CURRENT_AGENT_VERSION } from '../domain/agentVersion.js';
import { classifyPriority } from '../agent/classifier.js';
import { classifyUpdate, isUpgrade } from '../util/semver.js';
import { loadRepoConfig, isIgnored } from '../config/repoConfig.js';
import { detectEcosystems } from '../ecosystems/index.js';
import { now } from '../util/clock.js';
import { log } from '../util/logger.js';
import type { AnnealContext } from './context.js';

export interface ScanResult {
  created: DependencyUpdate[];
  skipped: number;
}

/**
 * T1/T2: scan a repository, find upgrade candidates + advisories, and create
 * `detected` records idempotently. Duplicate prevention (NF-008/NF-021): an
 * update_key with an existing active record is skipped.
 */
export async function scanRepository(
  ctx: AnnealContext,
  repoArg: string,
  repository?: string,
): Promise<ScanResult> {
  const repoPath = resolve(repoArg);
  const repoName = repository ?? deriveRepoName(repoPath);
  const repoConfig = await loadRepoConfig(repoPath);
  const ecosystems = await detectEcosystems(repoPath);

  if (ecosystems.length === 0) {
    log.warn('no supported manifest found', { repoPath });
    return { created: [], skipped: 0 };
  }

  const created: DependencyUpdate[] = [];
  let skipped = 0;

  for (const eco of ecosystems) {
    const deps = await eco.scan(repoPath);
    for (const dep of deps) {
      if (isIgnored(repoConfig, dep.name)) {
        skipped += 1;
        continue;
      }
      const latest = await ctx.metadata.latestVersion(
        eco.id,
        dep.name,
        dep.currentVersion,
      );
      if (!latest || !isUpgrade(dep.currentVersion, latest)) {
        skipped += 1;
        continue;
      }

      const advisories = await ctx.metadata.advisories(
        eco.id,
        dep.name,
        dep.currentVersion,
      );
      const cve = advisories[0];
      const key = updateKey(repoName, dep.name, latest);

      if (await ctx.store.getActiveUpdate(key)) {
        skipped += 1;
        continue; // NF-008: do not create a duplicate active record
      }

      const updateType = classifyUpdate(dep.currentVersion, latest);
      const record: DependencyUpdate = {
        update_key: key,
        repository: repoName,
        repoPath,
        ecosystem: eco.id,
        package_name: dep.name,
        current_version: dep.currentVersion,
        target_version: latest,
        update_type: updateType,
        is_dev_dependency: dep.isDev,
        priority: classifyPriority(updateType, dep.isDev, cve),
        risk_level: 'low',
        status: 'detected',
        agent_version: CURRENT_AGENT_VERSION,
        cve,
        history: [],
        created_at: now(),
        updated_at: now(),
      };
      await ctx.store.putUpdate(record);
      created.push(record);

      await ctx.notifier.notify({
        level: cve ? 'priority' : 'info',
        title: cve
          ? `Security update available: ${dep.name} (${cve.id})`
          : `Update available: ${dep.name}`,
        body: `${dep.currentVersion} → ${latest} (${updateType}, priority=${record.priority})`,
      });
      log.step('detected', { key, priority: record.priority });
    }
  }

  return { created, skipped };
}

function deriveRepoName(repoPath: string): string {
  const parts = repoPath.split(/[/\\]/).filter(Boolean);
  const last = parts[parts.length - 1] ?? 'repo';
  return `local/${last}`;
}
