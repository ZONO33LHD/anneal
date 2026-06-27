import type { AgentImprovement } from '../domain/improvement.js';
import { CURRENT_AGENT_VERSION } from '../domain/agentVersion.js';
import type { Store } from '../store/store.js';
import type { Llm } from '../providers/llm/llm.js';
import type { Notifier } from '../providers/notify/notifier.js';
import { now } from '../util/clock.js';
import { genId } from '../util/ids.js';
import { log } from '../util/logger.js';

/**
 * Decide whether the Annealing Loop should fire: the rolling average of the
 * last N finalized evaluations dropped below threshold.
 */
export async function shouldFire(
  store: Store,
  window: number,
  threshold: number,
): Promise<{ fire: boolean; average: number; sampled: number }> {
  const evals = (await store.listEvaluations())
    .filter((e) => e.score_status === 'final')
    .sort((a, b) => a.seq - b.seq);
  const recent = evals.slice(-window);
  if (recent.length === 0) return { fire: false, average: 0, sampled: 0 };
  const average =
    recent.reduce((sum, e) => sum + e.total_score, 0) / recent.length;
  return { fire: average < threshold, average: Math.round(average * 10) / 10, sampled: recent.length };
}

/**
 * Run one Annealing iteration: read failure cases, ask the LLM for an
 * improvement hypothesis and a concrete prompt/rule change, persist the
 * candidate, and notify. Adoption via A/B is out of scope for this build.
 */
export async function runAnnealing(
  store: Store,
  llm: Llm,
  notifier: Notifier,
  average: number,
): Promise<AgentImprovement | undefined> {
  const failures = await store.listFailures();
  if (failures.length === 0) {
    log.info('annealing: no failure cases to learn from');
    return undefined;
  }

  const digest = failures
    .slice(-10)
    .map(
      (f) =>
        `- ${f.update_key} score=${f.total_score} ${JSON.stringify(f.snapshot)}`,
    )
    .join('\n');

  const hypothesis = await llm.generate({
    prompt: `[hypothesis] Given these low-scoring dependency-update cases, propose the single most likely systemic weakness in the agent's judgement:\n${digest}`,
    temperature: 0.4,
  });

  const proposedChange = await llm.generate({
    prompt: `[prompt-improvement] Based on this hypothesis, propose one concrete change to the agent's risk/decision prompt to fix it:\nHypothesis: ${hypothesis}`,
    temperature: 0.4,
  });

  const candidateVersion = nextVersion(CURRENT_AGENT_VERSION);
  const improvement: AgentImprovement = {
    improvement_id: genId('imp'),
    trigger: `rolling avg ${average} below threshold`,
    target: 'prompt',
    previous_version: CURRENT_AGENT_VERSION,
    candidate_version: candidateVersion,
    hypothesis,
    proposedChange,
    status: 'candidate',
    created_at: now(),
  };
  await store.putImprovement(improvement);

  await notifier.notify({
    level: 'success',
    title: '🔥 Annealing Loop produced an improvement candidate',
    body:
      `Trigger: ${improvement.trigger}\n` +
      `Hypothesis: ${hypothesis}\n` +
      `Proposed (${candidateVersion}): ${proposedChange}`,
  });

  log.step('annealing: candidate generated', { id: improvement.improvement_id });
  return improvement;
}

function nextVersion(current: string): string {
  const m = current.match(/^(.*?)(\d+)$/);
  if (!m) return `${current}_v2`;
  return `${m[1]}${Number(m[2]) + 1}`;
}
