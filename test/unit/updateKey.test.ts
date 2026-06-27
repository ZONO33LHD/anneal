import { describe, it, expect } from 'vitest';
import { updateKey } from '../../src/domain/dependencyUpdate.js';

describe('updateKey', () => {
  it('is stable for the same inputs', () => {
    const a = updateKey('acme/repo', 'axios', '1.7.0');
    const b = updateKey('acme/repo', 'axios', '1.7.0');
    expect(a).toBe(b);
  });

  it('differs by target version', () => {
    expect(updateKey('acme/repo', 'axios', '1.7.0')).not.toBe(
      updateKey('acme/repo', 'axios', '1.8.0'),
    );
  });

  it('normalizes casing and separators', () => {
    expect(updateKey('Acme/Repo', 'AXIOS', '1.7.0')).toBe(
      'acme/repo::axios::1.7.0',
    );
  });
});
