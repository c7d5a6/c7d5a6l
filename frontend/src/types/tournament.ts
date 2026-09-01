import { apiUrl } from '../lib/api'

export type Participant = {
  name: string | null
  link: string | null
  race: string | null
  excluded: boolean
  /** Group standings winner / advancing player. */
  isWinner?: boolean
}

/** Scheduled or completed match between two sides. */
export type Result = {
  played: boolean
  scoreA: number | null
  scoreB: number | null
  participantA: Participant | null
  participantB: Participant | null
  dateTime: string | null
  stage: string | null
  phase?: string
  round?: string
  groupId?: number | null
  order: number
}

/** Named player pool within a tournament phase. */
export type TournamentGroup = {
  id?: number
  name: string
  phase: string
  sortOrder: number
  players: Participant[]
}

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
  groups: TournamentGroup[]
  finished: boolean | null
}

export type PlayerPage = {
  link: string
  name: string | null
  realName: string | null
  ids: string[]
  preferredRace: string | null
  /** Liquipedia source image URL (not for browser <img>). */
  portraitUrl: string | null
  /** True when a portrait blob is cached in the DB. */
  hasPortrait: boolean
  raceElos?: { race: string; elo: number }[]
}

export type PlayerRaceEntry = {
  playerRaceId: number
  playerId: number
  link: string
  name: string | null
  realName: string | null
  preferredRace: string | null
  hasPortrait: boolean
  race: string
  /** Stored player_race.elo (admin baseline). */
  elo: number
  /** Live season rating from match results, when a season is active. */
  projectedElo?: number | null
  seasonStartElo?: number | null
  rankDelta?: number | null
}

export type SeasonSummary = {
  id: number
  name: string
  status: string
  startedAt: string
  closedAt?: string | null
  readyToClose: boolean
}

/** Local API URL for a cached player portrait, or null. */
export function playerPortraitSrc(player: Pick<PlayerPage, 'link' | 'hasPortrait'>): string | null {
  if (!player.hasPortrait) return null
  return apiUrl(`/api/players/portrait?link=${encodeURIComponent(player.link)}`)
}

export type PlayerFieldChange = {
  field: string
  before: unknown
  after: unknown
}

export type PlayerSync = {
  exists: boolean
  same: boolean
  action: 'add' | 'update' | 'none'
  stored?: PlayerPage
  changes?: PlayerFieldChange[]
}

export type TournamentPlayerStatus = {
  name: string | null
  link: string | null
  race: string | null
  excluded: boolean
  inDatabase: boolean
  willImport: boolean
  importPending?: boolean
  skipReason?: string | null
}

export type TournamentSync = {
  exists: boolean
  same: boolean
  action: 'add' | 'update' | 'none'
  stored?: TournamentPage
  changes?: PlayerFieldChange[]
  players?: TournamentPlayerStatus[]
}

export type ParsePageType = 'tournament' | 'player' | 'unknown'

export type ParseResponse = {
  message: string
  pageType: ParsePageType
  tournament?: TournamentPage
  tournamentSync?: TournamentSync
  player?: PlayerPage
  playerSync?: PlayerSync
}

export type SavePlayerResponse = {
  message: string
  player: PlayerPage
  playerSync: PlayerSync
}

export type SaveTournamentResponse = {
  message: string
  tournament: TournamentPage
  tournamentSync: TournamentSync
  importQueued?: number
}

export type ErrorResponse = {
  error: string
}

export type AdminTournamentTab =
  | 'queue'
  | 'ongoing'
  | 'parsed'
  | 'finished'
  | 'ignored'
  | 'fantasy'
  | 'all'

export type AdminTournamentFlag = 'queue' | 'ongoing' | 'parsed' | 'finished' | 'ignored' | 'fantasy'

export type AdminTournament = {
  queueId: number | null
  tournamentId: number | null
  link: string
  name: string | null
  startDate: string | null
  endDate: string | null
  liquipediaTier: string | null
  section?: string | null
  disabled: boolean
  finished: boolean | null
  fantasyLeagueId: number | null
  flags: AdminTournamentFlag[]
}

export type AdminTournamentList = {
  items: AdminTournament[]
  total: number
  page: number
  pageSize: number
}

export function displayValue(value: string | number | boolean | null | undefined): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'boolean') return value ? 'yes' : 'no'
  return String(value)
}

export function displayChangeValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  if (Array.isArray(value)) return value.length ? value.join(', ') : '—'
  return String(value)
}
