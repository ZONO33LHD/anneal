import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname } from 'node:path';
import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import type { AgentEvaluation } from '../domain/evaluation.js';
import type { AgentImprovement, FailureCase } from '../domain/improvement.js';
import { isActive } from '../domain/states.js';
import type { Store } from './store.js';

interface Snapshot {
  updates: Record<string, DependencyUpdate>;
  evaluations: Record<string, AgentEvaluation>;
  failures: FailureCase[];
  improvements: AgentImprovement[];
}

function emptySnapshot(): Snapshot {
  return { updates: {}, evaluations: {}, failures: [], improvements: [] };
}

/**
 * Local JSON-backed store — the single source of truth for the demo/local mode.
 * Designed to be swapped for a Firestore implementation behind the same `Store`
 * interface (6.2). All reads/writes go through an in-memory snapshot that is
 * flushed to disk after each mutation, keeping the process stateless-friendly.
 */
export class JsonStore implements Store {
  private snapshot: Snapshot = emptySnapshot();
  private loaded = false;

  constructor(private readonly path: string) {}

  private async ensureLoaded(): Promise<void> {
    if (this.loaded) return;
    this.loaded = true;
    if (!this.path) return; // in-memory mode
    try {
      const raw = await readFile(this.path, 'utf8');
      this.snapshot = { ...emptySnapshot(), ...(JSON.parse(raw) as Snapshot) };
    } catch (err) {
      if ((err as NodeJS.ErrnoException).code !== 'ENOENT') throw err;
      this.snapshot = emptySnapshot();
    }
  }

  private async flush(): Promise<void> {
    if (!this.path) return; // in-memory mode
    await mkdir(dirname(this.path), { recursive: true });
    await writeFile(this.path, JSON.stringify(this.snapshot, null, 2), 'utf8');
  }

  async getUpdate(updateKey: string): Promise<DependencyUpdate | undefined> {
    await this.ensureLoaded();
    return this.snapshot.updates[updateKey];
  }

  async getActiveUpdate(updateKey: string): Promise<DependencyUpdate | undefined> {
    const record = await this.getUpdate(updateKey);
    return record && isActive(record.status) ? record : undefined;
  }

  async putUpdate(record: DependencyUpdate): Promise<void> {
    await this.ensureLoaded();
    this.snapshot.updates[record.update_key] = record;
    await this.flush();
  }

  async listUpdates(): Promise<DependencyUpdate[]> {
    await this.ensureLoaded();
    return Object.values(this.snapshot.updates);
  }

  async listActiveUpdates(): Promise<DependencyUpdate[]> {
    const all = await this.listUpdates();
    return all.filter((u) => isActive(u.status));
  }

  async getEvaluation(updateKey: string): Promise<AgentEvaluation | undefined> {
    await this.ensureLoaded();
    return this.snapshot.evaluations[updateKey];
  }

  async putEvaluation(record: AgentEvaluation): Promise<void> {
    await this.ensureLoaded();
    this.snapshot.evaluations[record.update_key] = record;
    await this.flush();
  }

  async listEvaluations(): Promise<AgentEvaluation[]> {
    await this.ensureLoaded();
    return Object.values(this.snapshot.evaluations);
  }

  async putFailure(record: FailureCase): Promise<void> {
    await this.ensureLoaded();
    this.snapshot.failures.push(record);
    await this.flush();
  }

  async listFailures(): Promise<FailureCase[]> {
    await this.ensureLoaded();
    return [...this.snapshot.failures];
  }

  async putImprovement(record: AgentImprovement): Promise<void> {
    await this.ensureLoaded();
    this.snapshot.improvements.push(record);
    await this.flush();
  }

  async listImprovements(): Promise<AgentImprovement[]> {
    await this.ensureLoaded();
    return [...this.snapshot.improvements];
  }
}

/** In-memory store for tests (no disk I/O). */
export class MemoryStore extends JsonStore {
  constructor() {
    super('');
  }

  // Override disk operations with no-ops by short-circuiting load/flush.
  // We reuse JsonStore logic but never touch the filesystem.
}
