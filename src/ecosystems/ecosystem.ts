import type { Ecosystem as EcosystemId } from '../domain/dependencyUpdate.js';

/** A dependency discovered in a manifest. */
export interface DetectedDependency {
  name: string;
  currentVersion: string;
  isDev: boolean;
  ecosystem: EcosystemId;
}

/**
 * An ecosystem knows how to read a manifest and apply a version bump to the
 * relevant files (F-003, F-013〜F-014). "What is the latest version" and "is
 * there a CVE" are intentionally NOT here — those come from a MetadataSource so
 * the file-handling logic stays deterministic and unit-testable.
 */
export interface Ecosystem {
  readonly id: EcosystemId;
  /** True if this ecosystem's manifest exists in the repo. */
  detect(repoPath: string): Promise<boolean>;
  /** Parse the manifest into a flat dependency list. */
  scan(repoPath: string): Promise<DetectedDependency[]>;
  /**
   * Bump `name` to `targetVersion`, returning the list of files changed.
   * Mutates files on disk (the working copy is a branch checkout in real mode).
   */
  applyUpdate(repoPath: string, name: string, targetVersion: string): Promise<string[]>;
}
