import { describe, it, expect } from 'vitest';
import { classifyPriority, isSecurityUpdate } from '../../src/agent/classifier.js';
import type { CveInfo } from '../../src/domain/dependencyUpdate.js';

const criticalCve: CveInfo = {
  id: 'CVE-1',
  severity: 'critical',
  affectedRange: '<1',
  patchedVersion: '1',
};

describe('classifier', () => {
  it('prioritizes critical CVEs', () => {
    expect(classifyPriority('patch', false, criticalCve)).toBe('critical');
  });

  it('treats exploitable CVEs as critical regardless of severity', () => {
    expect(
      classifyPriority('patch', false, { ...criticalCve, severity: 'low', exploitAvailable: true }),
    ).toBe('critical');
  });

  it('rates high CVEs as high', () => {
    expect(classifyPriority('patch', false, { ...criticalCve, severity: 'high' })).toBe('high');
  });

  it('rates dev dependencies low', () => {
    expect(classifyPriority('minor', true)).toBe('low');
  });

  it('defers majors to low priority', () => {
    expect(classifyPriority('major', false)).toBe('low');
  });

  it('rates ordinary patch/minor as medium', () => {
    expect(classifyPriority('patch', false)).toBe('medium');
    expect(classifyPriority('minor', false)).toBe('medium');
  });

  it('detects security updates', () => {
    expect(isSecurityUpdate(criticalCve)).toBe(true);
    expect(isSecurityUpdate(undefined)).toBe(false);
  });
});
