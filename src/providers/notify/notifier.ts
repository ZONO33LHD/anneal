/** Notification levels map to spec needs: priority for CVEs (F-019/F-050),
 * approval for human gate (F-055), success for improvements (F-054). */
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
