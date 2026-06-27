import type { DependencyUpdate } from '../src/domain/dependencyUpdate.js';
import { updateKey } from '../src/domain/dependencyUpdate.js';
import { CURRENT_AGENT_VERSION } from '../src/domain/agentVersion.js';

/** Build a DependencyUpdate for tests. Depends only on the domain layer. */
export function makeUpdate(overrides: Partial<DependencyUpdate> = {}): DependencyUpdate {
  const repository = overrides.repository ?? 'acme/demo';
  const pkg = overrides.package_name ?? 'axios';
  const target = overrides.target_version ?? '1.7.0';
  return {
    update_key: updateKey(repository, pkg, target),
    repository,
    ecosystem: 'npm',
    package_name: pkg,
    current_version: '1.6.2',
    target_version: target,
    update_type: 'minor',
    is_dev_dependency: false,
    priority: 'medium',
    risk_level: 'low',
    status: 'detected',
    agent_version: CURRENT_AGENT_VERSION,
    history: [],
    created_at: '2026-01-01T00:00:00.000Z',
    updated_at: '2026-01-01T00:00:00.000Z',
    ...overrides,
  };
}
