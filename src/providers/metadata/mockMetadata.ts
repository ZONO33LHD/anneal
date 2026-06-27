import type { CveInfo, Ecosystem } from '../../domain/dependencyUpdate.js';
import type { MetadataSource } from './metadata.js';

interface Entry {
  latest: string;
  advisories?: CveInfo[];
}

/**
 * Offline dataset used for deterministic local demos. Mirrors the dependencies
 * baked into `fixtures/sample-repo`. Keyed by `${ecosystem}:${name}`.
 */
const DATASET: Record<string, Entry> = {
  'npm:lodash': {
    latest: '4.17.21',
    advisories: [
      {
        id: 'CVE-2021-23337',
        severity: 'high',
        affectedRange: '<4.17.21',
        patchedVersion: '4.17.21',
        exploitAvailable: true,
        summary: 'Command injection via template in lodash.',
      },
    ],
  },
  'npm:axios': { latest: '1.7.0' },
  'npm:express': { latest: '4.19.2' },
  'npm:chalk': { latest: '5.3.0' }, // major: ESM-only, breaking
  'npm:typescript': { latest: '5.6.2' },
  'go:github.com/gin-gonic/gin': { latest: 'v1.9.1' },
  'go:golang.org/x/crypto': {
    latest: 'v0.17.0',
    advisories: [
      {
        id: 'CVE-2023-48795',
        severity: 'moderate',
        affectedRange: '<v0.17.0',
        patchedVersion: 'v0.17.0',
        summary: 'Terrapin attack on SSH transport (golang.org/x/crypto/ssh).',
      },
    ],
  },
};

export class MockMetadataSource implements MetadataSource {
  readonly name = 'mock';

  async latestVersion(
    ecosystem: Ecosystem,
    name: string,
  ): Promise<string | undefined> {
    return DATASET[`${ecosystem}:${name}`]?.latest;
  }

  async advisories(ecosystem: Ecosystem, name: string): Promise<CveInfo[]> {
    return DATASET[`${ecosystem}:${name}`]?.advisories ?? [];
  }
}
