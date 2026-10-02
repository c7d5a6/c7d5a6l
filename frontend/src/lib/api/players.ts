import { authFetch } from '../auth'
import { readApiError } from './http'
import type { PlayerDetail } from '../../types/tournament'

export type MergeCandidate = {
  playerId: number
  link: string
  name: string | null
  realName: string | null
  aliases: string[]
  score: number
  matchReason: string
}

export async function fetchPlayer(playerId: number): Promise<PlayerDetail> {
  const res = await authFetch(`/api/players/${playerId}`)
  if (!res.ok) {
    throw new Error(await readApiError(res, `player uplink failed (${res.status})`))
  }
  const data = (await res.json()) as { player?: PlayerDetail }
  if (!data.player) throw new Error('player uplink returned empty payload')
  return data.player
}

export async function fetchMergeCandidates(
  playerId: number,
  query = '',
  limit = 15,
): Promise<MergeCandidate[]> {
  const params = new URLSearchParams()
  if (query.trim()) params.set('q', query.trim())
  if (limit > 0) params.set('limit', String(limit))
  const qs = params.toString()
  const res = await authFetch(
    `/api/players/${playerId}/merge-candidates${qs ? `?${qs}` : ''}`,
  )
  if (!res.ok) {
    throw new Error(await readApiError(res, `merge candidates failed (${res.status})`))
  }
  const data = (await res.json()) as { candidates?: MergeCandidate[] }
  return data.candidates ?? []
}

export async function mergePlayers(mainPlayerId: number, aliasPlayerId: number): Promise<void> {
  const res = await authFetch('/api/players/merge', {
    method: 'POST',
    body: JSON.stringify({ mainPlayerId, aliasPlayerId }),
  })
  if (!res.ok) {
    throw new Error(await readApiError(res, `merge failed (${res.status})`))
  }
}
