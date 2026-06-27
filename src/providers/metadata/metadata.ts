import type { CveInfo, Ecosystem } from '../../domain/dependencyUpdate.js';

/**
 * Provides "what is the newest version" (F-001/F-003) and "is there an advisory"
 * (F-002/F-005). Split from Ecosystem so the registry/advisory I/O can be mocked
 * for offline, deterministic demos while ecosystems stay pure file logic.
 */
export interface MetadataSource {
  readonly name: string;
  latestVersion(
    ecosystem: Ecosystem,
    name: string,
    current: string,
  ): Promise<string | undefined>;
  advisories(ecosystem: Ecosystem, name: string, current: string): Promise<CveInfo[]>;
}
