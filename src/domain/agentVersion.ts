/**
 * Agent version. Every pull request and evaluation is tagged with the version of
 * the prompt / tool / rule set that produced it, so improvement effects can be
 * attributed and audited later.
 */
export type AgentVersionKind = 'prompt' | 'tool' | 'rule' | 'weights';

export interface AgentVersion {
  version_id: string;
  kind: AgentVersionKind;
  content_ref: string;
  active: boolean;
  created_at: string;
}

/** The composite version string stamped onto records, e.g. `prompt_v1`. */
export const CURRENT_AGENT_VERSION = 'prompt_v1';
