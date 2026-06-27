/**
 * Minimal structured logger. The orchestrator emits one line per state-machine
 * step so the asynchronous lifecycle is observable during demos.
 */
export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'step';

const ICONS: Record<LogLevel, string> = {
  debug: '·',
  info: 'ℹ',
  warn: '⚠',
  error: '✖',
  step: '→',
};

let verbose = process.env.ANNEAL_DEBUG === '1';

export function setVerbose(value: boolean): void {
  verbose = value;
}

function emit(level: LogLevel, msg: string, meta?: Record<string, unknown>): void {
  if (level === 'debug' && !verbose) return;
  const parts = [ICONS[level], msg];
  if (meta && Object.keys(meta).length > 0) {
    parts.push(
      Object.entries(meta)
        .map(([k, v]) => `${k}=${typeof v === 'string' ? v : JSON.stringify(v)}`)
        .join(' '),
    );
  }
  const line = parts.join(' ');
  if (level === 'error' || level === 'warn') {
    console.error(line);
  } else {
    console.log(line);
  }
}

export const log = {
  debug: (msg: string, meta?: Record<string, unknown>) => emit('debug', msg, meta),
  info: (msg: string, meta?: Record<string, unknown>) => emit('info', msg, meta),
  warn: (msg: string, meta?: Record<string, unknown>) => emit('warn', msg, meta),
  error: (msg: string, meta?: Record<string, unknown>) => emit('error', msg, meta),
  step: (msg: string, meta?: Record<string, unknown>) => emit('step', msg, meta),
};
