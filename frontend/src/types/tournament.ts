export type Participant = {
  name: string | null
}

/** Placeholder for future placement / bracket outcomes. */
export type Result = Record<string, never>

export type PlayerCounts = {
  total: number | null
  protoss: number | null
  zerg: number | null
  terran: number | null
}

export type TournamentPage = {
  link: string
  name: string | null
  startDate: string | null
  endDate: string | null
  liquipediaTier: string | null
  playerCounts: PlayerCounts | null
  participants: Participant[]
  results: Result[]
  finished: boolean | null
}

export type ParseResponse = {
  message: string
  tournament: TournamentPage
}

export type ErrorResponse = {
  error: string
}

export function displayValue(value: string | number | boolean | null | undefined): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'boolean') return value ? 'yes' : 'no'
  return String(value)
}
