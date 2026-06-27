import type { DependencyUpdate } from '../domain/dependencyUpdate.js';

/** Packages whose updates always need a human (auth/payment/security). */
const SENSITIVE_KEYWORDS = [
  'auth',
  'passport',
  'jsonwebtoken',
  'jwt',
  'bcrypt',
  'crypto',
  'stripe',
  'payment',
  'oauth',
  'session',
  'helmet',
];

export interface Decision {
  decision: 'auto' | 'human';
  reasons: string[];
}

function isSensitive(name: string): boolean {
  const lower = name.toLowerCase();
  return SENSITIVE_KEYWORDS.some((k) => lower.includes(k));
}

/**
 * Encodes the auto-vs-human decision flow. Returns whether the PR can proceed
 * automatically toward CI/review, or must wait for explicit human approval.
 * Low-quality model output is caught here and routed to a human as a safety net.
 */
export function decide(update: DependencyUpdate): Decision {
  const reasons: string[] = [];
  const impact = update.impact;

  // Sensitive packages: always human, even for security fixes.
  if (isSensitive(update.package_name)) {
    reasons.push('sensitive package (auth/payment/security)');
    return { decision: 'human', reasons };
  }

  // Security path: critical CVE is top priority; auto unless breaking.
  if (update.cve) {
    if (impact?.hasBreakingChange) {
      reasons.push(`security fix for ${update.cve.id} but breaking change`);
      return { decision: 'human', reasons };
    }
    reasons.push(`security fix for ${update.cve.id}, no breaking change`);
    return { decision: 'auto', reasons };
  }

  // Major updates always need a human.
  if (update.update_type === 'major') {
    reasons.push('major version bump');
    return { decision: 'human', reasons };
  }

  // Patch / dev dependency: safe to automate.
  if (update.update_type === 'patch' || update.is_dev_dependency) {
    reasons.push('patch or dev-only dependency');
    return maybeLowConfidence(update, reasons);
  }

  // Minor: depends on impact size.
  if (update.update_type === 'minor') {
    if (impact && impact.riskLevel === 'high') {
      reasons.push('minor bump with large/high-risk impact');
      return { decision: 'human', reasons };
    }
    reasons.push('minor bump with contained impact');
    return maybeLowConfidence(update, reasons);
  }

  reasons.push('default to human for safety');
  return { decision: 'human', reasons };
}

/** Final gate: low AI confidence or repeated CI failures route to a human. */
function maybeLowConfidence(update: DependencyUpdate, reasons: string[]): Decision {
  if (update.impact && update.impact.confidence < 0.5) {
    reasons.push(`low AI confidence (${update.impact.confidence.toFixed(2)})`);
    return { decision: 'human', reasons };
  }
  if ((update.ci?.attempts ?? 0) >= 2) {
    reasons.push('CI failed multiple times');
    return { decision: 'human', reasons };
  }
  return { decision: 'auto', reasons };
}
