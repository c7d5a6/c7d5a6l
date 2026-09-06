/** Trim a player link; Liquipedia page titles are case-sensitive (TerrOr ≠ Terror). */
export function normPlayerLink(link: string | null | undefined): string | null {
  const t = link?.trim()
  return t || null
}

/** Match key for a player race row (case-sensitive link, normalized race). */
export function playerRaceKey(link: string, race: string): string {
  return `${link.trim()}\0${race.trim().toLowerCase()}`
}
