import { authFetch } from '../auth'
import { readApiError } from './http'

export type SeasonSummary = {
  id: number
  name: string
  status: string
  startedAt: string
  closedAt?: string | null
  readyToClose: boolean
}

export type Season = SeasonSummary & {
  closingFantasyLeagueId?: number | null
  closingFantasyLeagueName?: string | null
}

export type ClosePreviewTournament = {
  id: number
  link: string
  name: string | null
  startDate: string | null
  endDate: string | null
  finished: boolean
  selected: boolean
  isFantasySource: boolean
}

export type SeasonClosePreview = {
  season: Season
  tournaments: ClosePreviewTournament[]
  closingFantasyLeagueId?: number | null
}

export async function fetchCurrentSeason(): Promise<Season | null> {
  const res = await authFetch('/api/seasons/current')
  if (res.status === 404) return null
  if (!res.ok) throw new Error(await readApiError(res, `season uplink failed (${res.status})`))
  const data = (await res.json()) as { season: Season }
  return data.season ?? null
}

export async function fetchSeasonClosePreview(): Promise<SeasonClosePreview> {
  const res = await authFetch('/api/seasons/close-preview')
  if (!res.ok) throw new Error(await readApiError(res, `season preview failed (${res.status})`))
  return (await res.json()) as SeasonClosePreview
}

export async function closeSeason(tournamentIds: number[]): Promise<Season> {
  const res = await authFetch('/api/seasons/close', {
    method: 'POST',
    body: JSON.stringify({ tournamentIds }),
  })
  if (!res.ok) throw new Error(await readApiError(res, `close season failed (${res.status})`))
  const data = (await res.json()) as { season: Season }
  return data.season
}
