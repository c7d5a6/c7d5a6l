import { authFetch } from '../auth'
import { readApiError } from './http'

export type MergeCandidate = {
  playerId: number
  link: string
  name: string | null
  realName: string | null
  aliases: string[]
  score: number
  matchReason: string
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
