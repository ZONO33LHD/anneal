import 'dotenv/config';
import type { Store } from '../store/store.js';
import { JsonStore } from '../store/jsonStore.js';
import type { Llm } from '../providers/llm/llm.js';
import { MockLlm } from '../providers/llm/mockLlm.js';
import { GeminiLlm } from '../providers/llm/geminiLlm.js';
import type { GitProvider } from '../providers/git/gitProvider.js';
import { MockGitProvider } from '../providers/git/mockGitProvider.js';
import { GitHubProvider } from '../providers/git/githubProvider.js';
import type { Notifier } from '../providers/notify/notifier.js';
import { ConsoleNotifier } from '../providers/notify/consoleNotifier.js';
import { SlackNotifier } from '../providers/notify/slackNotifier.js';
import type { MetadataSource } from '../providers/metadata/metadata.js';
import { MockMetadataSource } from '../providers/metadata/mockMetadata.js';
import { RealMetadataSource } from '../providers/metadata/realMetadata.js';
import { log } from '../util/logger.js';

export interface Settings {
  storePath: string;
  improveWindow: number;
  lowScoreThreshold: number;
}

export interface AnnealConfig {
  store: Store;
  llm: Llm;
  git: GitProvider;
  notifier: Notifier;
  metadata: MetadataSource;
  settings: Settings;
}

export interface LoadOptions {
  /** Force every provider to its mock/offline form (used by `demo`). */
  forceMock?: boolean;
  /** Override the store path (tests / demo isolation). */
  storePath?: string;
}

function num(name: string, fallback: number): number {
  const v = Number(process.env[name]);
  return Number.isFinite(v) && v > 0 ? v : fallback;
}

/**
 * Build the runtime context. Each capability independently picks its real
 * implementation when the relevant secret is present, otherwise the mock — so a
 * partial configuration still runs end-to-end (a key design goal).
 */
export function loadConfig(opts: LoadOptions = {}): AnnealConfig {
  const mock = opts.forceMock === true;
  const storePath =
    opts.storePath ?? process.env.ANNEAL_STORE_PATH ?? '.anneal/store.json';

  const llm: Llm =
    !mock && process.env.GEMINI_API_KEY
      ? new GeminiLlm(process.env.GEMINI_API_KEY)
      : new MockLlm();

  const git: GitProvider =
    !mock && process.env.GITHUB_TOKEN
      ? new GitHubProvider(process.env.GITHUB_TOKEN)
      : new MockGitProvider();

  const notifier: Notifier =
    !mock && process.env.SLACK_WEBHOOK_URL
      ? new SlackNotifier(process.env.SLACK_WEBHOOK_URL)
      : new ConsoleNotifier();

  const metadata: MetadataSource = mock
    ? new MockMetadataSource()
    : process.env.GEMINI_API_KEY || process.env.GITHUB_TOKEN
      ? new RealMetadataSource()
      : new MockMetadataSource();

  log.debug('providers selected', {
    llm: llm.name,
    git: git.name,
    notify: notifier.name,
    metadata: metadata.name,
  });

  return {
    store: new JsonStore(storePath),
    llm,
    git,
    notifier,
    metadata,
    settings: {
      storePath,
      improveWindow: num('ANNEAL_IMPROVE_WINDOW', 5),
      lowScoreThreshold: num('ANNEAL_LOW_SCORE_THRESHOLD', 70),
    },
  };
}
