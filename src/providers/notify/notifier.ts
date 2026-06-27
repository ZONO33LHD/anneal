/** Notification levels: priority for CVEs, approval for the human gate,
 * success for improvements. */
export type NotifyLevel = 'info' | 'priority' | 'approval' | 'success';

export interface NotifyMessage {
  level: NotifyLevel;
  title: string;
  body: string;
  url?: string;
}

export interface Notifier {
  readonly name: string;
  notify(msg: NotifyMessage): Promise<void>;
}
