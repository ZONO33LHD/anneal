import { describe, it, expect, afterEach } from 'vitest';
import { rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { JsonStore, MemoryStore } from '../../src/store/jsonStore.js';
import { makeUpdate } from '../factories.js';

const storePath = join(tmpdir(), `anneal-test-${process.pid}.json`);

afterEach(async () => {
  await rm(storePath, { force: true });
});

describe('JsonStore', () => {
  it('persists and reloads updates from disk', async () => {
    const a = new JsonStore(storePath);
    await a.putUpdate(makeUpdate({ package_name: 'axios' }));
    const b = new JsonStore(storePath);
    const all = await b.listUpdates();
    expect(all).toHaveLength(1);
    expect(all[0]!.package_name).toBe('axios');
  });

  it('distinguishes active from terminal records', async () => {
    const store = new MemoryStore();
    const active = makeUpdate({ package_name: 'axios', status: 'pr_created' });
    const done = makeUpdate({ package_name: 'lodash', target_version: '5', status: 'done' });
    await store.putUpdate(active);
    await store.putUpdate(done);
    expect(await store.getActiveUpdate(active.update_key)).toBeDefined();
    expect(await store.getActiveUpdate(done.update_key)).toBeUndefined();
    expect(await store.listActiveUpdates()).toHaveLength(1);
  });

  it('stores evaluations, failures, and improvements', async () => {
    const store = new MemoryStore();
    await store.putFailure({
      update_key: 'k',
      agent_version: 'v1',
      total_score: 50,
      reason: 'low',
      snapshot: {},
      created_at: 'now',
    });
    expect(await store.listFailures()).toHaveLength(1);
  });
});
