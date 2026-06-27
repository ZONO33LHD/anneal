import { readFile, writeFile, access } from 'node:fs/promises';
import { join } from 'node:path';
import type { Ecosystem, DetectedDependency } from './ecosystem.js';

async function exists(p: string): Promise<boolean> {
  try {
    await access(p);
    return true;
  } catch {
    return false;
  }
}

/** Matches a require line: `\tgithub.com/foo/bar v1.2.3` (optionally `// indirect`). */
const REQUIRE_LINE = /^\s*([^\s]+)\s+(v\d[^\s]*)(\s*\/\/\s*indirect)?\s*$/;

export class GoEcosystem implements Ecosystem {
  readonly id = 'go' as const;

  async detect(repoPath: string): Promise<boolean> {
    return exists(join(repoPath, 'go.mod'));
  }

  async scan(repoPath: string): Promise<DetectedDependency[]> {
    const raw = await readFile(join(repoPath, 'go.mod'), 'utf8');
    const out: DetectedDependency[] = [];
    let inRequireBlock = false;
    for (const line of raw.split('\n')) {
      const trimmed = line.trim();
      if (trimmed.startsWith('require (')) {
        inRequireBlock = true;
        continue;
      }
      if (inRequireBlock && trimmed === ')') {
        inRequireBlock = false;
        continue;
      }

      let target = line;
      // Single-line form: `require github.com/foo/bar v1.2.3`
      if (trimmed.startsWith('require ') && !trimmed.startsWith('require (')) {
        target = trimmed.replace(/^require\s+/, '');
      } else if (!inRequireBlock) {
        continue;
      }

      const m = target.match(REQUIRE_LINE);
      if (m) {
        const isIndirect = Boolean(m[3]);
        out.push({
          name: m[1]!,
          currentVersion: m[2]!,
          isDev: isIndirect,
          ecosystem: 'go',
        });
      }
    }
    return out;
  }

  async applyUpdate(
    repoPath: string,
    name: string,
    targetVersion: string,
  ): Promise<string[]> {
    const modPath = join(repoPath, 'go.mod');
    const raw = await readFile(modPath, 'utf8');
    const target = targetVersion.startsWith('v') ? targetVersion : `v${targetVersion}`;
    let updated = false;
    const lines = raw.split('\n').map((line) => {
      const m = line.match(REQUIRE_LINE);
      const single = line.trim().replace(/^require\s+/, '');
      const sm = single.match(REQUIRE_LINE);
      if (m && m[1] === name) {
        updated = true;
        return line.replace(m[2]!, target);
      }
      if (line.trim().startsWith('require ') && sm && sm[1] === name) {
        updated = true;
        return line.replace(sm[2]!, target);
      }
      return line;
    });
    const changed: string[] = [];
    if (updated) {
      await writeFile(modPath, lines.join('\n'), 'utf8');
      changed.push('go.mod');
      if (await exists(join(repoPath, 'go.sum'))) changed.push('go.sum');
    }
    return changed;
  }
}
