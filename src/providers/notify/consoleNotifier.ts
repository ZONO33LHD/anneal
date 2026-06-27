import type { Notifier, NotifyMessage, NotifyLevel } from './notifier.js';

const BADGE: Record<NotifyLevel, string> = {
  info: '🔔 INFO',
  priority: '🚨 PRIORITY',
  approval: '🙋 APPROVAL NEEDED',
  success: '✅ SUCCESS',
};

/** Default notifier: prints a Slack-like card to the console (F-049〜F-055). */
export class ConsoleNotifier implements Notifier {
  readonly name = 'console';

  async notify(msg: NotifyMessage): Promise<void> {
    const lines = [
      '',
      `┌─ ${BADGE[msg.level]} ─ Anneal`,
      `│ ${msg.title}`,
      `│ ${msg.body}`,
    ];
    if (msg.url) lines.push(`│ ${msg.url}`);
    lines.push('└────────────────────────────');
    console.log(lines.join('\n'));
  }
}
