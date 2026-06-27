import { log } from '../../util/logger.js';
import type {
  CiCheck,
  CiCheckOptions,
  CreatePrOptions,
  GitProvider,
  PullRequestRef,
  PushFixOptions,
} from './gitProvider.js';

type Octokit = {
  rest: {
    pulls: {
      create(args: Record<string, unknown>): Promise<{ data: { number: number; html_url: string } }>;
    };
    repos: {
      get(args: Record<string, unknown>): Promise<{ data: { default_branch: string } }>;
    };
    checks: {
      listForRef(args: Record<string, unknown>): Promise<{
        data: { check_runs: { conclusion: string | null; name: string }[] };
      }>;
    };
  };
};

/**
 * Real GitHub implementation. `@octokit/rest` is an optional dependency, imported
 * dynamically so the mock path needs no install. NOTE: committing file contents
 * to a branch via the Git data API is intentionally left as a focused follow-up;
 * this class covers PR creation and CI status, which is what the demo asserts in
 * real mode. Branch/commit creation falls back to a logged no-op.
 */
export class GitHubProvider implements GitProvider {
  readonly name = 'github';
  private clientPromise: Promise<Octokit> | undefined;

  constructor(private readonly token: string) {}

  private async client(): Promise<Octokit> {
    if (!this.clientPromise) {
      this.clientPromise = import('@octokit/rest').then(
        (m) => new (m as unknown as { Octokit: new (o: unknown) => Octokit }).Octokit({
          auth: this.token,
        }),
      );
    }
    return this.clientPromise;
  }

  private split(repository: string): { owner: string; repo: string } {
    const [owner, repo] = repository.split('/');
    return { owner: owner ?? '', repo: repo ?? '' };
  }

  async createBranchAndPr(opts: CreatePrOptions): Promise<PullRequestRef> {
    const gh = await this.client();
    const { owner, repo } = this.split(opts.repository);
    log.warn('github: branch/commit creation is a no-op in this build', {
      branch: opts.branch,
      files: opts.changedFiles.join(','),
    });
    const { data } = await gh.rest.pulls.create({
      owner,
      repo,
      title: opts.title,
      body: opts.body,
      head: opts.branch,
      base: opts.base,
    });
    return { number: data.number, url: data.html_url };
  }

  async pushFix(opts: PushFixOptions): Promise<void> {
    log.warn('github: pushFix is a no-op in this build', { pr: opts.prNumber });
  }

  async checkCi(opts: CiCheckOptions): Promise<CiCheck> {
    const gh = await this.client();
    const { owner, repo } = this.split(opts.repository);
    const { data } = await gh.rest.checks.listForRef({
      owner,
      repo,
      ref: opts.branch,
    });
    const runs = data.check_runs ?? [];
    const failed = runs.filter(
      (r) => r.conclusion && r.conclusion !== 'success' && r.conclusion !== 'neutral',
    );
    if (failed.length === 0) return { passed: true };
    return {
      passed: false,
      logSummary: `Checks failed: ${failed.map((r) => r.name).join(', ')}`,
    };
  }
}
