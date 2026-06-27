/**
 * Tiny semver helpers — enough to classify update_type (F-004) without pulling a
 * full semver dependency. Handles common range prefixes (^ ~ = v >=).
 */
export type UpdateType = 'patch' | 'minor' | 'major';

export interface SemVer {
  major: number;
  minor: number;
  patch: number;
}

/** Strip range operators and `v` prefix, returning a bare version string. */
export function cleanVersion(raw: string): string {
  return raw.trim().replace(/^[\^~>=<v\s]+/, '').split('-')[0]!.split('+')[0]!;
}

export function parse(raw: string): SemVer | null {
  const cleaned = cleanVersion(raw);
  const m = cleaned.match(/^(\d+)\.(\d+)\.(\d+)/);
  if (!m) return null;
  return { major: Number(m[1]), minor: Number(m[2]), patch: Number(m[3]) };
}

/** -1 if a<b, 0 if equal, 1 if a>b. Unparseable versions sort as equal. */
export function compare(a: string, b: string): number {
  const pa = parse(a);
  const pb = parse(b);
  if (!pa || !pb) return 0;
  if (pa.major !== pb.major) return pa.major < pb.major ? -1 : 1;
  if (pa.minor !== pb.minor) return pa.minor < pb.minor ? -1 : 1;
  if (pa.patch !== pb.patch) return pa.patch < pb.patch ? -1 : 1;
  return 0;
}

/** Classify the jump from `current` to `target` per semver semantics. */
export function classifyUpdate(current: string, target: string): UpdateType {
  const c = parse(current);
  const t = parse(target);
  if (!c || !t) return 'patch';
  if (t.major > c.major) return 'major';
  if (t.major === c.major && t.minor > c.minor) return 'minor';
  return 'patch';
}

/** True when `target` is strictly newer than `current`. */
export function isUpgrade(current: string, target: string): boolean {
  return compare(target, current) > 0;
}
