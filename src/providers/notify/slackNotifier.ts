import { log } from '../../util/logger.js';
import type { Notifier, NotifyMessage, NotifyLevel } from './notifier.js';

const EMOJI: Record<NotifyLevel, string> = {
  info: ':bell:',
  priority: ':rotating_light:',
  approval: ':raising_hand:',
  success: ':white_check_mark:',
};

/** Posts to a Slack Incoming Webhook using global fetch (no extra deps). */
export class SlackNotifier implements Notifier {
  readonly name = 'slack';

  constructor(private readonly webhookUrl: string) {}

  async notify(msg: NotifyMessage): Promise<void> {
    const text = [
      `${EMOJI[msg.level]} *${msg.title}*`,
      msg.body,
      msg.url ? `<${msg.url}|Open>` : '',
    ]
      .filter(Boolean)
      .join('\n');
    try {
      const res = await fetch(this.webhookUrl, {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ text }),
      });
      if (!res.ok) log.warn('Slack webhook returned non-OK', { status: res.status });
    } catch (err) {
      log.warn('Slack notify failed', { err: String(err) });
    }
  }
}
