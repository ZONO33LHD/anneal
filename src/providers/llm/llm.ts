/**
 * LLM abstraction. The default model is the cheapest tier; quality wobble is
 * backstopped by deterministic rules + human approval. The agent uses the LLM
 * only for prose/enrichment (PR bodies, log summaries, improvement hypotheses)
 * so the pipeline still works end-to-end with the mock.
 */
export interface LlmRequest {
  system?: string;
  prompt: string;
  temperature?: number;
  maxTokens?: number;
}

export interface Llm {
  readonly name: string;
  readonly model: string;
  generate(req: LlmRequest): Promise<string>;
}
