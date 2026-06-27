/**
 * Lightweight, dependency-free id helpers.
 *
 * We avoid `Math.random`-only ids so that ids stay readable in demo logs while
 * remaining unique enough for a single local process.
 */
let counter = 0;

/** Monotonic counter combined with time, e.g. `run_lm0k3f-7`. */
export function genId(prefix: string): string {
  counter += 1;
  const stamp = Date.now().toString(36);
  return `${prefix}_${stamp}-${counter}`;
}

/** Monotonic sequence number — stable ordering for events within a process. */
let seq = 0;
export function nextSeq(): number {
  seq += 1;
  return seq;
}

/** Stable slug from arbitrary text (used inside update_key). */
export function slug(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9._/-]+/g, '-')
    .replace(/^-+|-+$/g, '');
}
