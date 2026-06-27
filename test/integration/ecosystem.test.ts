import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { cp, rm, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { NpmEcosystem } from '../../src/ecosystems/npmEcosystem.js';
import { GoEcosystem } from '../../src/ecosystems/goEcosystem.js';
import { detectEcosystems } from '../../src/ecosystems/index.js';

const fixture = join(process.cwd(), 'fixtures', 'sample-repo');
let work: string;

beforeEach(async () => {
  work = join(tmpdir(), `anneal-eco-${process.pid}-${Math.floor(performance.now())}`);
  await cp(fixture, work, { recursive: true });
});

afterEach(async () => {
  await rm(work, { recursive: true, force: true });
});

describe('npm ecosystem', () => {
  it('parses dependencies and devDependencies', async () => {
    const deps = await new NpmEcosystem().scan(work);
    const names = deps.map((d) => d.name);
    expect(names).toContain('lodash');
    expect(names).toContain('axios');
    expect(deps.find((d) => d.name === 'typescript')!.isDev).toBe(true);
  });

  it('applies an update preserving range prefix', async () => {
    const changed = await new NpmEcosystem().applyUpdate(work, 'axios', '1.7.0');
    expect(changed).toContain('package.json');
    const pkg = JSON.parse(await readFile(join(work, 'package.json'), 'utf8'));
    expect(pkg.dependencies.axios).toBe('^1.7.0');
  });
});

describe('go ecosystem', () => {
  it('parses require block', async () => {
    const deps = await new GoEcosystem().scan(work);
    const names = deps.map((d) => d.name);
    expect(names).toContain('github.com/gin-gonic/gin');
    expect(names).toContain('golang.org/x/crypto');
  });

  it('applies an update to go.mod', async () => {
    const changed = await new GoEcosystem().applyUpdate(
      work,
      'github.com/gin-gonic/gin',
      'v1.9.1',
    );
    expect(changed).toContain('go.mod');
    const mod = await readFile(join(work, 'go.mod'), 'utf8');
    expect(mod).toContain('gin v1.9.1');
  });
});

describe('detection', () => {
  it('detects both npm and go', async () => {
    const found = await detectEcosystems(work);
    expect(found.map((e) => e.id).sort()).toEqual(['go', 'npm']);
  });
});
