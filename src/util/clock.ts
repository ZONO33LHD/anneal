/** Current timestamp as ISO string. Wrapped so tests can stub it if needed. */
export function now(): string {
  return new Date().toISOString();
}
