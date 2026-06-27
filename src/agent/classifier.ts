import type { CveInfo, Priority } from '../domain/dependencyUpdate.js';
import type { UpdateType } from '../util/semver.js';

/**
 * Update priority classification:
 *  🔴 Critical: exploitable CVE / exploit published
 *  🟠 High:     high+ CVE
 *  🟡 Medium:   ordinary patch / minor
 *  ⚪ Low:      dev-only, or major with breaking-change concern
 */
export function classifyPriority(
  updateType: UpdateType,
  isDev: boolean,
  cve?: CveInfo,
): Priority {
  if (cve) {
    if (cve.severity === 'critical' || cve.exploitAvailable) return 'critical';
    if (cve.severity === 'high') return 'high';
    return 'high'; // any advisory is at least High priority to triage
  }
  if (isDev) return 'low';
  if (updateType === 'major') return 'low'; // defer; breaking-change risk
  return 'medium';
}

/** True when the update is security-driven (changes PR semantics + notify level). */
export function isSecurityUpdate(cve?: CveInfo): boolean {
  return Boolean(cve);
}
