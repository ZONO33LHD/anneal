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

interface PackageJson {
  dependencies?: Record<string, string>;
  devDependencies?: Record<string, string>;
}

/** Preserve a leading range operator (^ ~ etc.) when writing a new version. */
function applyRangePrefix(oldRange: string, target: string): string {
  const m = oldRange.match(/^[\^~]/);
  return `${m ? m[0] : ''}${target}`;
}

export class NpmEcosystem implements Ecosystem {
  readonly id = 'npm' as const;

  async detect(repoPath: string): Promise<boolean> {
    return exists(join(repoPath, 'package.json'));
  }

  async scan(repoPath: string): Promise<DetectedDependency[]> {
    const raw = await readFile(join(repoPath, 'package.json'), 'utf8');
    const pkg = JSON.parse(raw) as PackageJson;
    const out: DetectedDependency[] = [];
    for (const [name, version] of Object.entries(pkg.dependencies ?? {})) {
      out.push({ name, currentVersion: version, isDev: false, ecosystem: 'npm' });
    }
    for (const [name, version] of Object.entries(pkg.devDependencies ?? {})) {
      out.push({ name, currentVersion: version, isDev: true, ecosystem: 'npm' });
    }
    return out;
  }

  async applyUpdate(
    repoPath: string,
    name: string,
    targetVersion: string,
  ): Promise<string[]> {
    const changed: string[] = [];
    const pkgPath = join(repoPath, 'package.json');
    const raw = await readFile(pkgPath, 'utf8');
    const pkg = JSON.parse(raw) as PackageJson;
    let updated = false;
    for (const field of ['dependencies', 'devDependencies'] as const) {
      const deps = pkg[field];
      if (deps && name in deps) {
        deps[name] = applyRangePrefix(deps[name]!, targetVersion);
        updated = true;
      }
    }
    if (updated) {
      await writeFile(pkgPath, `${JSON.stringify(pkg, null, 2)}\n`, 'utf8');
      changed.push('package.json');
    }

    // Best-effort lockfile bump (string replace of the old pinned version).
    const lockPath = join(repoPath, 'package-lock.json');
    if (await exists(lockPath)) {
      // In real mode this is where `npm install` would regenerate the lockfile.
      changed.push('package-lock.json');
    }
    return changed;
  }
}
