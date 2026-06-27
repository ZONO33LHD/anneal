import { log } from '../../util/logger.js';
import type { Llm, LlmRequest } from './llm.js';

/**
 * Real Gemini implementation. `@google/genai` is an optional dependency and is
 * imported dynamically so the mock path works without it installed (NF: offline
 * E2E). Defaults to the cheapest model per spec 6.1.
 */
export class GeminiLlm implements Llm {
  readonly name = 'gemini';

  constructor(
    private readonly apiKey: string,
    readonly model: string = process.env.ANNEAL_LLM_MODEL ?? 'gemini-2.5-flash-lite',
  ) {}

  async generate(req: LlmRequest): Promise<string> {
    try {
      const mod = (await import('@google/genai')) as unknown as {
        GoogleGenAI: new (opts: { apiKey: string }) => {
          models: {
            generateContent(args: {
              model: string;
              contents: string;
              config?: Record<string, unknown>;
            }): Promise<{ text?: string }>;
          };
        };
      };
      const client = new mod.GoogleGenAI({ apiKey: this.apiKey });
      const contents = req.system ? `${req.system}\n\n${req.prompt}` : req.prompt;
      const res = await client.models.generateContent({
        model: this.model,
        contents,
        config: {
          temperature: req.temperature ?? 0.2,
          maxOutputTokens: req.maxTokens ?? 1024,
        },
      });
      return res.text ?? '';
    } catch (err) {
      log.warn('Gemini call failed; returning empty completion', {
        err: String(err),
      });
      return '';
    }
  }
}
