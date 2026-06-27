import { describe, it, expect } from 'vitest';
import { computeComponents, computeTotal } from '../../src/scoring/scorer.js';
import { buildEvaluation } from '../../src/scoring/partialScore.js';
import { SCORE_WEIGHTS } from '../../src/domain/evaluation.js';
import { makeUpdate } from '../factories.js';

describe('scorer', () => {
  it('scores a clean pass highly', () => {
    const u = makeUpdate({
      status: 'merged',
      ci: { status: 'passed', attempts: 0 },
      reviewCommentCount: 0,
      impact: {
        usageSites: ['a'],
        hasBreakingChange: false,
        affectedFiles: ['a'],
        riskLevel: 'low',
        confidence: 0.9,
        summary: 's',
      },
    });
    const c = computeComponents(u);
    expect(c.ci).toBe(100);
    expect(c.merge).toBe(100);
    expect(computeTotal(c)).toBeGreaterThan(90);
  });

  it('penalizes failed CI', () => {
    const u = makeUpdate({ status: 'closed', ci: { status: 'failed', attempts: 0 } });
    const c = computeComponents(u);
    expect(c.ci).toBe(20);
    expect(c.merge).toBe(0);
    expect(computeTotal(c)).toBeLessThan(70);
  });

  it('discounts pass-after-fix', () => {
    const u = makeUpdate({ ci: { status: 'passed', attempts: 2 } });
    expect(computeComponents(u).ci).toBe(70);
  });

  it('weights sum to 1', () => {
    const sum = Object.values(SCORE_WEIGHTS).reduce((a, b) => a + b, 0);
    expect(sum).toBeCloseTo(1, 5);
  });

  it('builds partial evaluations before merge and final after', () => {
    const partial = buildEvaluation(makeUpdate({ status: 'ci_passed' }));
    expect(partial.score_status).toBe('partial');
    const final = buildEvaluation(makeUpdate({ status: 'merged' }), partial);
    expect(final.score_status).toBe('final');
    expect(final.run_id).toBe(partial.run_id); // run_id preserved across updates
  });
});
