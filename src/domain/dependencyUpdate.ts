import type { State } from './states.js';
import { slug } from '../util/ids.js';
import type { UpdateType } from '../util/semver.js';

export type Ecosystem = 'npm' | 'go';
export type RiskLevel = 'low' | 'medium' | 'high';
export type Priority = 'critical' | 'high' | 'medium' | 'low';

/** CVE / advisory information attached to a security-driven update (F-005). */
export interface CveInfo {
  id: string;
  severity: 'critical' | 'high' | 'moderate' | 'low';
  affectedRange: string;
  patchedVersion: string;
  exploitAvailable?: boolean;
  summary?: string;
}

/** Output of the impact analysis stage (F-007〜F-012). */
export interface ImpactAnalysis {
  usageSites: string[];
  hasBreakingChange: boolean;
  migrationGuideUrl?: string;
  affectedFiles: string[];
  riskLevel: RiskLevel;
  summary: string;
  confidence: number; // 0..1, AI Confidence (10章)
}

/** Latest CI outcome for the PR (F-021〜F-026). */
export interface CiResult {
  status: 'running' | 'passed' | 'failed';
  failureCategory?:
    | 'Dependency Conflict'
    | 'API Breaking Change'
    | 'Test Update Required'
    | 'Environment Issue'
    | 'Unknown';
  fixable?: boolean;
  attempts: number;
  logSummary?: string;
}

/** One entry in the immutable transition history (NF-005 auditability). */
export interface TransitionLog {
  from: State;
  to: State;
  reason: string;
  at: string;
}

/**
 * The central aggregate. `update_key` (7.1) makes a given update unique and is
 * the basis for idempotency / duplicate prevention (NF-008, NF-021).
 */
export interface DependencyUpdate {
  update_key: string;
  repository: string;
  /** Local working-copy path for this repo (local mode). */
  repoPath?: string;
  ecosystem: Ecosystem;
  package_name: string;
  current_version: string;
  target_version: string;
  update_type: UpdateType;
  is_dev_dependency: boolean;
  priority: Priority;
  risk_level: RiskLevel;
  status: State;
  agent_version: string;
  cve?: CveInfo;
  impact?: ImpactAnalysis;
  ci?: CiResult;
  pull_request_url?: string;
  pull_request_number?: number;
  branch?: string;
  reviewCommentCount?: number;
  history: TransitionLog[];
  created_at: string;
  updated_at: string;
}

/** Build the canonical unique key: repository + package + target_version. */
export function updateKey(
  repository: string,
  packageName: string,
  targetVersion: string,
): string {
  return `${slug(repository)}::${slug(packageName)}::${slug(targetVersion)}`;
}
