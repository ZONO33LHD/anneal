/** A file change to include in a branch/commit. */
export interface FileChange {
  path: string;
  /** New full contents (real mode commits these). */
  content?: string;
}

export interface CreatePrOptions {
  repository: string;
  base: string;
  branch: string;
  title: string;
  body: string;
  changedFiles: string[];
}

export interface PushFixOptions {
  repository: string;
  branch: string;
  prNumber: number;
  message: string;
  changedFiles: string[];
}

export interface CiCheckOptions {
  repository: string;
  prNumber: number;
  branch: string;
  /** Attempt index (0 = initial run). Drives the mock self-heal scenario. */
  attempt: number;
  /** Mock-only hint: simulate a fixable failure on the first attempt. */
  shouldFailFirst?: boolean;
}

export interface CiCheck {
  passed: boolean;
  logSummary?: string;
}

export interface PullRequestRef {
  number: number;
  url: string;
}

/**
 * Git hosting abstraction (F-013〜F-017, F-021〜F-022). Real mode talks to GitHub
 * via Octokit; mock mode returns scripted results so the whole lifecycle runs
 * locally without a token.
 */
export interface GitProvider {
  readonly name: string;
  createBranchAndPr(opts: CreatePrOptions): Promise<PullRequestRef>;
  pushFix(opts: PushFixOptions): Promise<void>;
  checkCi(opts: CiCheckOptions): Promise<CiCheck>;
}
