import type { Llm, LlmRequest } from './llm.js';

/**
 * Deterministic mock LLM. Produces plausible, task-aware text by recognising a
 * tag we put at the start of each prompt (e.g. `[pr-body]`, `[ci-summary]`).
 * Keeps demos reproducible and tests free of network/keys.
 */
export class MockLlm implements Llm {
  readonly name = 'mock';
  readonly model = 'mock-flash-lite';

  async generate(req: LlmRequest): Promise<string> {
    const tag = req.prompt.match(/^\[([a-z-]+)\]/)?.[1];
    switch (tag) {
      case 'pr-body':
        return 'This update keeps the dependency current and pulls in upstream fixes. Risk is contained to the documented usage sites; tests cover the affected paths.';
      case 'ci-summary':
        return 'The build failed because the bumped package changed an exported signature; the call site needs a small adjustment. This looks mechanically fixable.';
      case 'hypothesis':
        return 'Risk for minor bumps of widely-imported packages is being underestimated, leading to surprise CI failures. The classifier should weight import breadth more heavily.';
      case 'prompt-improvement':
        return 'Add an explicit instruction: "When >10 import sites exist, raise risk one level and require a usage-diff before auto-PR." This should reduce false LOW-risk calls.';
      default:
        return 'OK';
    }
  }
}
