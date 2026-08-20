import type { Result } from '../types/tournament'

/** Calendar day YYYY-MM-DD in UTC from an RFC3339 timestamp. */
export function utcDay(iso: string | null | undefined): string | null {
  if (!iso) return null
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) {
    const m = iso.match(/^(\d{4}-\d{2}-\d{2})/)
    return m ? m[1] : null
  }
  return d.toISOString().slice(0, 10)
}

export function isPlayoffsPhase(phase: string): boolean {
  return phase.toLowerCase().includes('playoff')
}

export function isFinalsRound(name: string): boolean {
  const n = name.toLowerCase()
  return n.includes('grand final') || n === 'finals' || n === 'final'
}

export type GroupWithResults<G extends { id?: number; phase: string; name: string }> = {
  group: G
  results: Result[]
}

export function resultsForGroup(
  results: Result[],
  groupId: number | undefined,
  phase: string,
  name: string,
): Result[] {
  return results
    .filter((r) => {
      if (groupId != null && r.groupId != null) return r.groupId === groupId
      return (
        (r.phase ?? '').toLowerCase() === phase.toLowerCase() &&
        (r.round ?? '').toLowerCase() === name.toLowerCase()
      )
    })
    .sort((a, b) => a.order - b.order)
}

export function partitionGroups<G extends { id?: number; phase: string; name: string; sortOrder: number }>(
  groups: G[],
  results: Result[],
  today: string,
): { upcoming: GroupWithResults<G>[]; completed: GroupWithResults<G>[] } {
  const upcoming: GroupWithResults<G>[] = []
  const completed: GroupWithResults<G>[] = []
  const sorted = [...groups].sort((a, b) => a.sortOrder - b.sortOrder)

  const finalsPlayed = sorted.some(
    (x) =>
      isPlayoffsPhase(x.phase) &&
      isFinalsRound(x.name) &&
      resultsForGroup(results, x.id, x.phase, x.name).some((r) => r.played),
  )

  for (const g of sorted) {
    const gr = resultsForGroup(results, g.id, g.phase, g.name)
    const entry = { group: g, results: gr }
    if (isPlayoffsPhase(g.phase)) {
      if (finalsPlayed) completed.push(entry)
      else upcoming.push(entry)
      continue
    }
    if (isPoolGroupUpcoming(gr, today)) upcoming.push(entry)
    else completed.push(entry)
  }
  return { upcoming, completed }
}

function isPoolGroupUpcoming(results: Result[], today: string): boolean {
  if (results.length === 0) return true
  const allPlayed = results.every((r) => r.played)
  if (!allPlayed) return true
  const days = results.map((r) => utcDay(r.dateTime)).filter(Boolean) as string[]
  if (days.length === 0) return false
  const latest = [...days].sort().at(-1)!
  return latest >= today
}

/** Today's matches, or nearest future day. */
export function pickDayMatches(
  results: Result[],
  today: string,
): { label: string; day: string; matches: Result[] } {
  const withDay = results
    .map((r) => ({ r, day: utcDay(r.dateTime) }))
    .filter((x): x is { r: Result; day: string } => Boolean(x.day))
    .sort((a, b) => a.day.localeCompare(b.day) || a.r.order - b.r.order)

  const todayMatches = withDay.filter((x) => x.day === today).map((x) => x.r)
  if (todayMatches.length) {
    return { label: 'Today', day: today, matches: todayMatches }
  }
  const future = withDay.filter((x) => x.day > today)
  if (future.length) {
    const day = future[0].day
    return {
      label: day,
      day,
      matches: future.filter((x) => x.day === day).map((x) => x.r),
    }
  }
  return { label: 'Today', day: today, matches: [] }
}

export function participantLinksInResults(results: Result[]): Set<string> {
  const links = new Set<string>()
  for (const r of results) {
    const a = r.participantA?.link?.trim().toLowerCase()
    const b = r.participantB?.link?.trim().toLowerCase()
    if (a) links.add(a)
    if (b) links.add(b)
  }
  return links
}
