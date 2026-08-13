import fantasyArt from '../assets/background/fantasy.webp'
import meArt from '../assets/background/me.webp'
import parserArt from '../assets/background/parser.webp'
import playersArt from '../assets/background/players.webp'
import { normalizePath } from './routes'

/**
 * Canonical stage art per route — STYLE_GUIDE §3a / BACKGROUND.md.
 * Distinct pages must not share the same file (even within one palette family).
 */
const STAGE_BACKGROUNDS: Record<string, string> = {
  '/parser': parserArt,
  '/players': playersArt,
  '/fantasy-league': fantasyArt,
  '/users': meArt,
  '/titles': meArt,
  '/me': meArt,
}

const DEFAULT_ART = fantasyArt

export function stageBackgroundForPath(path: string): string {
  const p = normalizePath(path)
  if (p.startsWith('/tournaments')) return STAGE_BACKGROUNDS['/fantasy-league']
  if (p.startsWith('/fantasy-manage')) return STAGE_BACKGROUNDS['/fantasy-league']
  return STAGE_BACKGROUNDS[p] ?? DEFAULT_ART
}
