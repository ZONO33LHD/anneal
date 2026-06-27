import type { CiResult } from '../domain/dependencyUpdate.js';
import type { Llm } from '../providers/llm/llm.js';

export type FailureCategory = NonNullable<CiResult['failureCategory']>;

interface Rule {
  category: FailureCategory;
  fixable: boolean;
  patterns: RegExp[];
}

/** Keyword rules for the failure categories. Checked in order. */
const RULES: Rule[] = [
  {
    category: 'Dependency Conflict',
    fixable: true,
    patterns: [/peer dep/i, /ERESOLVE/i, /version conflict/i, /incompatible/i],
  },
  {
    category: 'API Breaking Change',
    fixable: false,
    patterns: [/is not a function/i, /has no exported member/i, /breaking/i, /signature/i],
  },
  {
    category: 'Test Update Required',
    fixable: true,
    patterns: [/test.*failed/i, /assertion/i, /expected .* received/i, /snapshot/i],
  },
  {
    category: 'Environment Issue',
    fixable: false,
    patterns: [/ENOENT/i, /timeout/i, /network/i, /rate limit/i, /ECONNREFUSED/i],
  },
];

export interface CiFailureAnalysis {
  category: FailureCategory;
  fixable: boolean;
  summary: string;
}

/**
 * Classify a CI failure from its log summary. Deterministic keyword rules
 * decide category + fixability; the LLM provides a human-readable summary so
 * reviewers can quickly understand why CI broke.
 */
export async function classifyCiFailure(
  logSummary: string,
  llm: Llm,
): Promise<CiFailureAnalysis> {
  let matched: Rule | undefined;
  for (const rule of RULES) {
    if (rule.patterns.some((p) => p.test(logSummary))) {
      matched = rule;
      break;
    }
  }
  const category = matched?.category ?? 'Unknown';
  const fixable = matched?.fixable ?? false;

  const aiSummary = await llm.generate({
    prompt: `[ci-summary] Summarize this CI failure and whether it is mechanically fixable:\n${logSummary}`,
    temperature: 0.2,
  });

  return {
    category,
    fixable,
    summary: aiSummary && aiSummary !== 'OK' ? aiSummary : logSummary,
  };
}
