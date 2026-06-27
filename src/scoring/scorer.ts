import type { DependencyUpdate } from '../domain/dependencyUpdate.js';
import { SCORE_WEIGHTS } from '../domain/evaluation.js';

/** Individual component scores, each 0..100 (9.5). */
export interface ScoreComponents {
  ci: number;
  review: number;
  risk: number;
  prQuality: number;
  fixAccuracy: number;
  merge: number;
  regression: number;
}

function clamp(n: number): number {
  return Math.max(0, Math.min(100, Math.round(n)));
}

/** CI score: clean pass best, pass-after-fix discounted, failure poor. */
function ciScore(u: DependencyUpdate): number {
  const ci = u.ci;
  if (!ci) return 50; // unknown yet
  if (ci.status === 'passed') return ci.attempts > 1 ? 70 : 100;
  if (ci.status === 'failed') return 20;
  return 50;
}

/** Review burden from comment count (fewer is better). */
function reviewScore(u: DependencyUpdate): number {
  if (u.reviewCommentCount === undefined) return 70;
  return clamp(100 - u.reviewCommentCount * 15);
}

/** Risk prediction: did the predicted risk match the CI outcome? */
function riskScore(u: DependencyUpdate): number {
  const risk = u.impact?.riskLevel;
  const passed = u.ci?.status === 'passed';
  const cleanPass = passed && (u.ci?.attempts ?? 0) <= 1;
  if (!risk) return 60;
  if (risk === 'low') return cleanPass ? 100 : 40; // predicted easy, was it?
  if (risk === 'high') return passed ? 70 : 85; // correctly flagged trouble
  return 75; // medium
}

/** PR quality from completeness of analysis + body. */
function prQualityScore(u: DependencyUpdate): number {
  let score = 60;
  if (u.impact) score += 15;
  if (u.impact?.summary) score += 10;
  if (u.pull_request_url) score += 10;
  if (u.cve) score += 5;
  return clamp(score);
}

/** Fix accuracy: no fix needed is ideal; successful fix good; failed fix poor. */
function fixAccuracyScore(u: DependencyUpdate): number {
  const attempts = u.ci?.attempts ?? 0;
  if (attempts === 0) return 100;
  if (u.ci?.status === 'passed') return 80;
  return 30;
}

/** Merge outcome from terminal-ish status. */
function mergeScore(u: DependencyUpdate): number {
  switch (u.status) {
    case 'merged':
    case 'monitoring_regression':
    case 'done':
      return 100;
    case 'changes_requested':
      return 50;
    case 'closed':
    case 'regressed':
      return 0;
    default:
      return 50; // not decided yet
  }
}

/** Regression score (recorded separately; 100 unless a regression was found). */
function regressionScore(u: DependencyUpdate): number {
  if (u.status === 'regressed') return 0;
  if (u.status === 'done') return 100;
  return 90; // monitoring window not closed yet
}

export function computeComponents(u: DependencyUpdate): ScoreComponents {
  return {
    ci: ciScore(u),
    review: reviewScore(u),
    risk: riskScore(u),
    prQuality: prQualityScore(u),
    fixAccuracy: fixAccuracyScore(u),
    merge: mergeScore(u),
    regression: regressionScore(u),
  };
}

/** Weighted total (9.5). Regression is tracked separately, not in the total. */
export function computeTotal(c: ScoreComponents): number {
  const total =
    c.ci * SCORE_WEIGHTS.ci +
    c.review * SCORE_WEIGHTS.review +
    c.risk * SCORE_WEIGHTS.risk +
    c.prQuality * SCORE_WEIGHTS.prQuality +
    c.fixAccuracy * SCORE_WEIGHTS.fixAccuracy +
    c.merge * SCORE_WEIGHTS.merge;
  return Math.round(total * 10) / 10;
}
