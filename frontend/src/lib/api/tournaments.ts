import { authFetch } from '../auth'
import type {
  AdminTournamentList,
  AdminTournamentTab,
  SaveTournamentResponse,
  TournamentPage,
  TournamentSync,
} from '../../types/tournament'
import { readApiError } from './http'

export async function fetchAdminTournaments(
  tab: AdminTournamentTab,
  page: number,
  pageSize = 20,
): Promise<AdminTournamentList> {
  const q = new URLSearchParams({
    tab,
    page: String(page),
    pageSize: String(pageSize),
  })
  const res = await authFetch(`/api/tournament-queue?${q}`)
  if (!res.ok) throw new Error(await readApiError(res, `tournament list failed (${res.status})`))
  return (await res.json()) as AdminTournamentList
}

export async function syncTournamentQueue(): Promise<number> {
  const res = await authFetch('/api/tournament-queue/sync', { method: 'POST' })
  if (!res.ok) throw new Error(await readApiError(res, `listing sync failed (${res.status})`))
  const data = (await res.json()) as { count: number }
  return data.count
}

export async function parseQueueItem(id: number): Promise<SaveTournamentResponse> {
  const res = await authFetch(`/api/tournament-queue/${id}/parse`, { method: 'POST' })
  if (!res.ok) throw new Error(await readApiError(res, `parse failed (${res.status})`))
  return (await res.json()) as SaveTournamentResponse
}

export async function ignoreQueueItem(id: number): Promise<void> {
  const res = await authFetch(`/api/tournament-queue/${id}/ignore`, { method: 'POST' })
  if (!res.ok) throw new Error(await readApiError(res, `ignore failed (${res.status})`))
}

export async function fetchTournament(id: number): Promise<{
  message: string
  tournament: TournamentPage
  tournamentSync: TournamentSync
}> {
  const res = await authFetch(`/api/tournaments/${id}`)
  if (!res.ok) throw new Error(await readApiError(res, `tournament failed (${res.status})`))
  return (await res.json()) as {
    message: string
    tournament: TournamentPage
    tournamentSync: TournamentSync
  }
}
