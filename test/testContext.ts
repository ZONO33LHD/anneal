import type { AnnealContext } from '../src/orchestrator/context.js';
import { MemoryStore } from '../src/store/jsonStore.js';
import { MockLlm } from '../src/providers/llm/mockLlm.js';
import { MockGitProvider } from '../src/providers/git/mockGitProvider.js';
import { MockMetadataSource } from '../src/providers/metadata/mockMetadata.js';
import type { Notifier, NotifyMessage } from '../src/providers/notify/notifier.js';

/** Notifier that captures messages instead of printing (quiet tests). */
export class CapturingNotifier implements Notifier {
  readonly name = 'capture';
  readonly messages: NotifyMessage[] = [];
  async notify(msg: NotifyMessage): Promise<void> {
    this.messages.push(msg);
  }
}

/** Build an all-mock AnnealContext for integration/e2e tests. */
export function makeTestContext(
  overrides: Partial<AnnealContext> = {},
): AnnealContext {
  return {
    store: new MemoryStore(),
    llm: new MockLlm(),
    git: new MockGitProvider(),
    notifier: new CapturingNotifier(),
    metadata: new MockMetadataSource(),
    settings: { storePath: '', improveWindow: 5, lowScoreThreshold: 90 },
    simulate: true,
    ...overrides,
  };
}
