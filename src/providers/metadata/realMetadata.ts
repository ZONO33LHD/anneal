import type { CveInfo, Ecosystem } from '../../domain/dependencyUpdate.js';
import { cleanVersion } from '../../util/semver.js';
import { log } from '../../util/logger.js';
import type { MetadataSource } from './metadata.js';

/**
 * Live metadata using only Node's global `fetch` (no extra deps):
 *  - latest version from the npm registry / Go module proxy
 *  - advisories from the OSV.dev API
 * Network failures degrade gracefully to "no update / no advisory".
 */
export class RealMetadataSource implements MetadataSource {
  readonly name = 'osv+registry';

  async latestVersion(
    ecosystem: Ecosystem,
    name: string,
  ): Promise<string | undefined> {
    try {
      if (ecosystem === 'npm') {
        const res = await fetch(`https://registry.npmjs.org/${name}/latest`);
        if (!res.ok) return undefined;
        const body = (await res.json()) as { version?: string };
        return body.version;
      }
      // Go module proxy.
      const res = await fetch(
        `https://proxy.golang.org/${encodeURIComponent(name)}/@latest`,
      );
      if (!res.ok) return undefined;
      const body = (await res.json()) as { Version?: string };
      return body.Version;
    } catch (err) {
      log.warn('latestVersion lookup failed', { name, err: String(err) });
      return undefined;
    }
  }

  async advisories(
    ecosystem: Ecosystem,
    name: string,
    current: string,
  ): Promise<CveInfo[]> {
    try {
      const res = await fetch('https://api.osv.dev/v1/query', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          version: cleanVersion(current),
          package: { name, ecosystem: ecosystem === 'npm' ? 'npm' : 'Go' },
        }),
      });
      if (!res.ok) return [];
      const body = (await res.json()) as { vulns?: OsvVuln[] };
      return (body.vulns ?? []).map(toCveInfo);
    } catch (err) {
      log.warn('advisory lookup failed', { name, err: String(err) });
      return [];
    }
  }
}

interface OsvVuln {
  id: string;
  summary?: string;
  aliases?: string[];
  database_specific?: { severity?: string };
  affected?: { ranges?: { events?: { fixed?: string }[] }[] }[];
}

function toCveInfo(v: OsvVuln): CveInfo {
  const cve = v.aliases?.find((a) => a.startsWith('CVE-')) ?? v.id;
  const fixed = v.affected?.[0]?.ranges?.[0]?.events?.find((e) => e.fixed)?.fixed;
  const sev = (v.database_specific?.severity ?? 'moderate').toLowerCase();
  const severity = (['critical', 'high', 'moderate', 'low'].includes(sev)
    ? sev
    : 'moderate') as CveInfo['severity'];
  return {
    id: cve,
    severity,
    affectedRange: 'see advisory',
    patchedVersion: fixed ?? 'unknown',
    summary: v.summary,
  };
}
