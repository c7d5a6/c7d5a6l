import bg002 from '../../assets/background/lemon-sky-studios-lemon-sky-studios-002.webp'
import bg008 from '../../assets/background/lemon-sky-studios-lemon-sky-studios-008.webp'
import bg010 from '../../assets/background/lemon-sky-studios-lemon-sky-studios-010.webp'
import bg015 from '../../assets/background/lemon-sky-studios-lemon-sky-studios-015.webp'
import terranArt from '../../assets/background/lemon-sky-studios-lemonsky-studio-terran-01.webp'
import { normalizePath } from './routes'

/**
 * Canonical stage art per route — STYLE_GUIDE §3a / BACKGROUND.md.
 * Distinct pages must not share the same file (even within one palette family).
 */
const STAGE_BACKGROUNDS: Record<string, string> = {
  '/parser': bg002,
  '/players': bg008,
  '/fantasy-league': bg015,
  '/users': terranArt,
  '/titles': terranArt,
  '/me': bg010,
}

const DEFAULT_ART = bg015

export function stageBackgroundForPath(path: string): string {
  const p = normalizePath(path)
  if (p.startsWith('/tournaments')) return STAGE_BACKGROUNDS['/fantasy-league']
  if (p.startsWith('/fantasy-manage')) return STAGE_BACKGROUNDS['/fantasy-league']
  return STAGE_BACKGROUNDS[p] ?? DEFAULT_ART
}
