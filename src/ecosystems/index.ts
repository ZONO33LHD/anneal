import type { Ecosystem } from './ecosystem.js';
import { NpmEcosystem } from './npmEcosystem.js';
import { GoEcosystem } from './goEcosystem.js';

export const ECOSYSTEMS: Ecosystem[] = [new NpmEcosystem(), new GoEcosystem()];

/** Return every ecosystem whose manifest is present in the repo. */
export async function detectEcosystems(repoPath: string): Promise<Ecosystem[]> {
  const found: Ecosystem[] = [];
  for (const eco of ECOSYSTEMS) {
    if (await eco.detect(repoPath)) found.push(eco);
  }
  return found;
}

export function getEcosystem(id: string): Ecosystem | undefined {
  return ECOSYSTEMS.find((e) => e.id === id);
}

export * from './ecosystem.js';
