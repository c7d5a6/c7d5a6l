import type { UserTitle } from './user'

export type FantasyLeague = {
  id: number
  tournamentId: number
  tournamentLink: string
  tournamentName: string | null
  started: boolean
  finished: boolean
  maxPlayers: number
  maxCost: number
}

export type FantasyPlayerRow = {
  id: number
  fantasyLeagueId: number
  tournamentPlayerId: number
  name: string | null
  link: string | null
  race: string | null
  cost: number
  pointsRo24: number | null
  pointsRo16: number | null
  pointsRo8: number | null
  pointsRo4: number | null
  pointsRo2: number | null
  pointsEarned: number
  defeated: boolean
  isWinner: boolean
  elo?: number
}

export type FantasyPreviewPlayer = {
  tournamentPlayerId: number
  name: string | null
  link: string | null
  race: string | null
  elo: number
  cost: number
}

export type FantasyTeamMemberRow = {
  fantasyPlayerId: number
  name: string | null
  link: string | null
  race: string | null
  cost: number
  pointsEarned: number
  defeated: boolean
  isWinner: boolean
  elo: number
}

export type FantasyTeamRow = {
  id: number
  fantasyLeagueId: number
  userId: number
  userAlias: string
  rank: number
  points: number
  cost: number
  members: FantasyTeamMemberRow[]
  titles?: UserTitle[]
}

export type TournamentSummary = {
  id: number
  link: string
  name: string | null
}

export const POINT_STAGES = ['Ro24', 'Ro16', 'Ro8', 'Ro4', 'Ro2'] as const
export type PointStage = (typeof POINT_STAGES)[number]

export function stagePoints(p: FantasyPlayerRow, stage: PointStage): number | null {
  switch (stage) {
    case 'Ro24':
      return p.pointsRo24
    case 'Ro16':
      return p.pointsRo16
    case 'Ro8':
      return p.pointsRo8
    case 'Ro4':
      return p.pointsRo4
    case 'Ro2':
      return p.pointsRo2
  }
}

export function stageReached(p: FantasyPlayerRow): number {
  let n = 0
  for (const stage of POINT_STAGES) {
    if (stagePoints(p, stage) != null) n++
    else break
  }
  return n
}

export function sortByElo<T extends { elo?: number | null }>(rows: T[]): T[] {
  return [...rows].sort((a, b) => (b.elo ?? 0) - (a.elo ?? 0))
}
