import { authFetch } from '../auth'
import type {
  FantasyGroup,
  FantasyLeague,
  FantasyMatchBoard,
  FantasyPlayerRow,
  FantasyTeamRow,
} from '../../types/fantasy'
import { readApiError } from './http'

export async function fetchFantasyLeagues(): Promise<FantasyLeague[]> {
  const res = await authFetch('/api/fantasy-leagues')
  if (!res.ok) throw new Error(await readApiError(res, `leagues uplink failed (${res.status})`))
  const data = (await res.json()) as { leagues: FantasyLeague[] }
  return data.leagues ?? []
}

export async function fetchFantasyLeague(id: number): Promise<FantasyLeague> {
  const res = await authFetch(`/api/fantasy-leagues/${id}`)
  if (!res.ok) throw new Error(await readApiError(res, `league failed (${res.status})`))
  const data = (await res.json()) as { league: FantasyLeague }
  return data.league
}

export async function fetchActiveFantasyLeague(): Promise<FantasyLeague | null> {
  const res = await authFetch('/api/fantasy-leagues/active')
  if (res.status === 404) return null
  if (!res.ok) throw new Error(await readApiError(res, `fantasy league uplink failed (${res.status})`))
  const data = (await res.json()) as { league: FantasyLeague }
  return data.league ?? null
}

export async function fetchFantasyPlayers(
  leagueId: number,
  sort = 'elo',
): Promise<FantasyPlayerRow[]> {
  const res = await authFetch(`/api/fantasy-leagues/${leagueId}/players?sort=${sort}`)
  if (!res.ok) throw new Error(await readApiError(res, `fantasy players uplink failed (${res.status})`))
  const data = (await res.json()) as { players: FantasyPlayerRow[] }
  return data.players ?? []
}

export async function fetchFantasyTeams(leagueId: number): Promise<FantasyTeamRow[]> {
  const res = await authFetch(`/api/fantasy-leagues/${leagueId}/teams`)
  if (!res.ok) throw new Error(await readApiError(res, `fantasy teams uplink failed (${res.status})`))
  const data = (await res.json()) as { teams: FantasyTeamRow[] }
  return data.teams ?? []
}

export async function fetchFantasyGroups(leagueId: number): Promise<FantasyGroup[]> {
  const res = await authFetch(`/api/fantasy-leagues/${leagueId}/groups`)
  if (!res.ok) throw new Error(await readApiError(res, `groups uplink failed (${res.status})`))
  const data = (await res.json()) as { groups: FantasyGroup[] }
  return data.groups ?? []
}

export async function fetchFantasyMatchBoard(leagueId: number): Promise<FantasyMatchBoard> {
  const res = await authFetch(`/api/fantasy-leagues/${leagueId}/match-board`)
  if (!res.ok) throw new Error(await readApiError(res, `match board uplink failed (${res.status})`))
  return (await res.json()) as FantasyMatchBoard
}
