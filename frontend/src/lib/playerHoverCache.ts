import { authFetch } from './auth'
import { playerPortraitSrc, type PlayerPage } from '../types/tournament'

export type PlayerLookupResult =
  | { status: 'ok'; player: PlayerPage }
  | { status: 'missing' }
  | { status: 'error'; message: string }

const infoCache = new Map<string, PlayerLookupResult>()
const infoInflight = new Map<string, Promise<PlayerLookupResult>>()

function cacheKey(link: string): string {
  return link.trim()
}

/** One request per link; subsequent callers share the same promise / cache. */
export function loadPlayerInfo(link: string): Promise<PlayerLookupResult> {
  const key = cacheKey(link)
  const hit = infoCache.get(key)
  if (hit) return Promise.resolve(hit)

  const pending = infoInflight.get(key)
  if (pending) return pending

  const req = (async (): Promise<PlayerLookupResult> => {
    try {
      const res = await authFetch(`/api/players/lookup?link=${encodeURIComponent(key)}`)
      if (res.status === 404) {
        const miss: PlayerLookupResult = { status: 'missing' }
        infoCache.set(key, miss)
        return miss
      }
      if (!res.ok) {
        return { status: 'error', message: `lookup failed (${res.status})` }
      }
      const player = (await res.json()) as PlayerPage
      if (!player.ids) player.ids = []
      const ok: PlayerLookupResult = { status: 'ok', player }
      infoCache.set(key, ok)
      return ok
    } catch {
      return { status: 'error', message: 'uplink offline' }
    } finally {
      infoInflight.delete(key)
    }
  })()

  infoInflight.set(key, req)
  return req
}

/**
 * Portrait URL for <img src>. Do not fetch() this cross-origin — browsers block
 * blob reads without CORS, while an image element does not need CORS to display.
 */
export function loadPlayerPortrait(link: string, hasPortrait: boolean): Promise<string | null> {
  return Promise.resolve(playerPortraitSrc({ link: cacheKey(link), hasPortrait }))
}

export function peekPlayerInfo(link: string): PlayerLookupResult | undefined {
  return infoCache.get(cacheKey(link))
}

export function peekPlayerPortrait(link: string): string | undefined {
  const info = infoCache.get(cacheKey(link))
  if (!info || info.status !== 'ok' || !info.player.hasPortrait) return undefined
  return playerPortraitSrc({ link: cacheKey(link), hasPortrait: true }) ?? undefined
}

/** Drop cached lookup so the next hover refetch includes updated race elos. */
export function invalidatePlayerInfo(link: string): void {
  const key = cacheKey(link)
  infoCache.delete(key)
  infoInflight.delete(key)
}
