import type { JSX } from 'solid-js'
import { Show } from 'solid-js'
import { FantasyLeaguePage } from '../pages/FantasyLeaguePage'
import { FantasyManageDetailPage } from '../pages/FantasyManageDetailPage'
import { FantasyManagePage } from '../pages/FantasyManagePage'
import { MePage } from '../pages/MePage'
import { ParserPage } from '../pages/ParserPage'
import { PlayersPage } from '../pages/PlayersPage'
import { TitlesPage } from '../pages/TitlesPage'
import { UsersPage } from '../pages/UsersPage'
import { isAdmin } from './auth'
import {
  NAV_PATHS,
  fantasyManageLeagueId,
  isFantasyManagePath,
  normalizePath,
} from './routes'

/**
 * Paths registered with the Solid router (content is `() => null` —
 * StageShell owns the visible page stack for slide animations).
 */
export const LAYER_ROUTE_PATHS = [
  '/parser',
  '/players',
  '/fantasy-league',
  '/fantasy-manage',
  '/fantasy-manage/:id',
  '/users',
  '/titles',
  '/me',
] as const

/**
 * Resolve normalized path → page component.
 * Admin-only pages return null when `!isAdmin()` (guardAdminPath redirects).
 */
export function renderPageForPath(path: string): JSX.Element | null {
  const p = normalizePath(path)

  if (p === NAV_PATHS.parser) return isAdmin() ? <ParserPage /> : null
  if (p === NAV_PATHS.players) return <PlayersPage />
  if (p === NAV_PATHS.fantasy) return <FantasyLeaguePage />
  if (isFantasyManagePath(p)) {
    if (!isAdmin()) return null
    const id = fantasyManageLeagueId(p)
    return id != null ? <FantasyManageDetailPage leagueId={id} /> : <FantasyManagePage />
  }
  if (p === NAV_PATHS.users) return isAdmin() ? <UsersPage /> : null
  if (p === NAV_PATHS.titles) return <TitlesPage />
  if (p === '/me') return <MePage />
  return null
}

/** Idle uplink while auth boots — avoids blank admin StageShell flash. */
export function AuthBootStatus(): JSX.Element {
  return <p class="status status--idle">Locking uplink…</p>
}

export function PageLayerContent(props: { path: string; ready: boolean }): JSX.Element {
  return (
    <Show when={props.ready} fallback={<AuthBootStatus />}>
      {renderPageForPath(props.path)}
    </Show>
  )
}
