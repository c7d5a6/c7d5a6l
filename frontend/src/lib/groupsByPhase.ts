/** Group items by `phase` (empty → "—"). Shared by fantasy groups + tournament telemetry. */
export function groupsByPhase<T extends { phase: string }>(items: T[]): [string, T[]][] {
  const map = new Map<string, T[]>()
  for (const item of items) {
    const phase = item.phase || '—'
    const list = map.get(phase) ?? []
    list.push(item)
    map.set(phase, list)
  }
  return [...map.entries()]
}
