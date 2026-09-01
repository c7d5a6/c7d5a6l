import type { NavRailId } from '../components/NavRail'
import { homePath, isAdmin } from './auth'

export const NAV_PATHS: Record<NavRailId, string> = {
  parser: '/parser',
  tournaments: '/tournaments',
  players: '/players',
  fantasy: '/fantasy-league',
  leagues: '/fantasy-manage',
  users: '/users',
  titles: '/titles',
}

export function normalizePath(path: string): string {
  if (path === '/' || path === '') return homePath()
  const cleaned = path.replace(/\/+$/, '') || homePath()
  if (cleaned === '/me') return '/me'
  return cleaned
}

export function pathToNavId(path: string): NavRailId {
  const p = normalizePath(path)
  if (p.startsWith(NAV_PATHS.tournaments)) return 'tournaments'
  if (p.startsWith(NAV_PATHS.players)) return 'players'
  if (p.startsWith(NAV_PATHS.fantasy)) return 'fantasy'
  if (p.startsWith(NAV_PATHS.leagues)) return 'leagues'
  if (p.startsWith(NAV_PATHS.users)) return 'users'
  if (p.startsWith(NAV_PATHS.titles)) return 'titles'
  return 'parser'
}

export function isFantasyManagePath(path: string): boolean {
  const p = normalizePath(path)
  return p === NAV_PATHS.leagues || p.startsWith(`${NAV_PATHS.leagues}/`)
}

export function isTournamentsPath(path: string): boolean {
  const p = normalizePath(path)
  return p === NAV_PATHS.tournaments || p.startsWith(`${NAV_PATHS.tournaments}/`)
}

export function tournamentDetailId(path: string): number | null {
  const p = normalizePath(path)
  const prefix = `${NAV_PATHS.tournaments}/`
  if (!p.startsWith(prefix)) return null
  const id = Number(p.slice(prefix.length).split('/')[0])
  return Number.isFinite(id) && id > 0 ? id : null
}

export function fantasyManageLeagueId(path: string): number | null {
  const p = normalizePath(path)
  const prefix = `${NAV_PATHS.leagues}/`
  if (!p.startsWith(prefix)) return null
  const id = Number(p.slice(prefix.length).split('/')[0])
  return Number.isFinite(id) && id > 0 ? id : null
}

export function isSeasonClosePath(path: string): boolean {
  return normalizePath(path) === '/season-close'
}

/** Redirect non-admins away from admin-only routes. */
export function guardAdminPath(path: string): string | null {
  const p = normalizePath(path)
  if (
    (p === NAV_PATHS.parser ||
      p === NAV_PATHS.users ||
      isFantasyManagePath(p) ||
      isTournamentsPath(p) ||
      isSeasonClosePath(p)) &&
    !isAdmin()
  ) {
    return NAV_PATHS.fantasy
  }
  return null
}
