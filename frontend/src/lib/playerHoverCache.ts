import { authFetch } from './auth'
import { playerPortraitSrc, type PlayerPage } from '../types/tournament'

export type PlayerLookupResult =
  | { status: 'ok'; player: PlayerPage }
  | { status: 'missing' }
  | { status: 'error'; message: string }

const infoCache = new Map<string, PlayerLookupResult>()
const infoInflight = new Map<string, Promise<PlayerLookupResult>>()

/** object URL or API URL string, keyed by player link */
const portraitURLCache = new Map<string, string>()
const portraitInflight = new Map<string, Promise<string | null>>()

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
 * One portrait fetch per link. Caches a blob object URL for instant re-show.
 * Returns null when no portrait is available.
 */
export function loadPlayerPortrait(link: string, hasPortrait: boolean): Promise<string | null> {
  const key = cacheKey(link)
  if (!hasPortrait) return Promise.resolve(null)

  const hit = portraitURLCache.get(key)
  if (hit) return Promise.resolve(hit)

  const pending = portraitInflight.get(key)
  if (pending) return pending

  const apiSrc = playerPortraitSrc({ link: key, hasPortrait: true })
  if (!apiSrc) return Promise.resolve(null)

  const req = (async (): Promise<string | null> => {
    try {
      const res = await fetch(apiSrc)
      if (!res.ok) return null
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      portraitURLCache.set(key, url)
      return url
    } catch {
      return null
    } finally {
      portraitInflight.delete(key)
    }
  })()

  portraitInflight.set(key, req)
  return req
}

export function peekPlayerInfo(link: string): PlayerLookupResult | undefined {
  return infoCache.get(cacheKey(link))
}

export function peekPlayerPortrait(link: string): string | undefined {
  return portraitURLCache.get(cacheKey(link))
}

/** Drop cached lookup so the next hover refetch includes updated race elos. */
export function invalidatePlayerInfo(link: string): void {
  const key = cacheKey(link)
  infoCache.delete(key)
  infoInflight.delete(key)
}
