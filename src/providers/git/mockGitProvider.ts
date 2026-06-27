import { log } from '../../util/logger.js';
import { slug } from '../../util/ids.js';
import type {
  CiCheck,
  CiCheckOptions,
  CreatePrOptions,
  GitProvider,
  PullRequestRef,
  PushFixOptions,
} from './gitProvider.js';

/**
 * Scripted git provider for local demos. PR numbers are derived from the branch
 * so they are stable across stateless ticks. CI outcome is deterministic:
 * when `shouldFailFirst` is set, attempt 0 fails (fixable) and later attempts
 * pass — reproducing the "CI fails → self-heal → passes" story.
 */
export class MockGitProvider implements GitProvider {
  readonly name = 'mock';
  private prCounter = 1000;

  async createBranchAndPr(opts: CreatePrOptions): Promise<PullRequestRef> {
    this.prCounter += 1;
    const number = this.prCounter;
    const url = `https://github.com/${opts.repository}/pull/${number}`;
    log.info('mock: opened PR', { branch: opts.branch, url });
    return { number, url };
  }

  async pushFix(opts: PushFixOptions): Promise<void> {
    log.info('mock: pushed fix commit', {
      pr: opts.prNumber,
      files: opts.changedFiles.join(','),
    });
  }

  async checkCi(opts: CiCheckOptions): Promise<CiCheck> {
    if (opts.shouldFailFirst && opts.attempt === 0) {
      return {
        passed: false,
        logSummary: `npm test failed: 2 tests broke after the bump (${slug(
          opts.branch,
        )}).`,
      };
    }
    return { passed: true };
  }
}
