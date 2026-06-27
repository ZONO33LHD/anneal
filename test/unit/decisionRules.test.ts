import { describe, it, expect } from 'vitest';
import { decide } from '../../src/agent/decisionRules.js';
import { makeUpdate } from '../factories.js';
import type { ImpactAnalysis } from '../../src/domain/dependencyUpdate.js';

const lowImpact: ImpactAnalysis = {
  usageSites: ['a.ts'],
  hasBreakingChange: false,
  affectedFiles: ['a.ts'],
  riskLevel: 'low',
  confidence: 0.9,
  summary: 's',
};

describe('decision rules (10章)', () => {
  it('auto-approves patch updates', () => {
    const d = decide(makeUpdate({ update_type: 'patch', impact: lowImpact }));
    expect(d.decision).toBe('auto');
  });

  it('requires human for major updates', () => {
    const d = decide(makeUpdate({ update_type: 'major' }));
    expect(d.decision).toBe('human');
  });

  it('requires human for sensitive packages', () => {
    const d = decide(makeUpdate({ package_name: 'passport', update_type: 'patch' }));
    expect(d.decision).toBe('human');
  });

  it('auto-approves non-breaking security fixes', () => {
    const d = decide(
      makeUpdate({
        update_type: 'patch',
        impact: lowImpact,
        cve: { id: 'CVE-X', severity: 'high', affectedRange: '<1', patchedVersion: '1' },
      }),
    );
    expect(d.decision).toBe('auto');
  });

  it('requires human for breaking security fixes', () => {
    const d = decide(
      makeUpdate({
        update_type: 'minor',
        impact: { ...lowImpact, hasBreakingChange: true },
        cve: { id: 'CVE-X', severity: 'high', affectedRange: '<1', patchedVersion: '1' },
      }),
    );
    expect(d.decision).toBe('human');
  });

  it('requires human for high-risk minor bumps', () => {
    const d = decide(
      makeUpdate({ update_type: 'minor', impact: { ...lowImpact, riskLevel: 'high' } }),
    );
    expect(d.decision).toBe('human');
  });

  it('routes low-confidence updates to a human', () => {
    const d = decide(
      makeUpdate({ update_type: 'patch', impact: { ...lowImpact, confidence: 0.3 } }),
    );
    expect(d.decision).toBe('human');
  });
});
