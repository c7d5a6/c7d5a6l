import { For, Show, createMemo, type JSX } from 'solid-js'
import { isAdmin } from '../lib/auth'

export type NavRailId = 'parser' | 'players' | 'fantasy' | 'leagues' | 'users' | 'titles'

export type NavRailProps = {
  /** When false, rail is not rendered. */
  visible?: boolean
  /** Controlled active id from the router. */
  activeId?: NavRailId
  onSelect?: (id: NavRailId) => void
}

const ALL_ITEMS: { id: NavRailId; lines: string[]; adminOnly?: boolean }[] = [
  { id: 'parser', lines: ['Parser'], adminOnly: true },
  { id: 'players', lines: ['Players'] },
  { id: 'fantasy', lines: ['Fantasy', 'League'] },
  { id: 'leagues', lines: ['Leagues'], adminOnly: true },
  { id: 'users', lines: ['Users'], adminOnly: true },
  { id: 'titles', lines: ['Titles'], adminOnly: true },
]

/**
 * Left command rail (desktop) / bottom strip (mobile).
 * Selection is controlled by the router via `activeId` + `onSelect`.
 */
export function NavRail(props: NavRailProps): JSX.Element {
  const visible = () => props.visible !== false
  const items = createMemo(() => ALL_ITEMS.filter((item) => !item.adminOnly || isAdmin()))
  const activeId = () => props.activeId ?? (isAdmin() ? 'parser' : 'fantasy')

  function select(id: NavRailId) {
    props.onSelect?.(id)
  }

  return (
    <Show when={visible()}>
      <nav class="nav-rail" aria-label="Command channels">
        <div class="nav-rail__cap" aria-hidden="true" />
        <ul class="nav-rail__list">
          <For each={items()}>
            {(item) => {
              const active = () => activeId() === item.id
              return (
                <li class="nav-rail__item">
                  <button
                    type="button"
                    classList={{
                      'nav-rail__btn': true,
                      'nav-rail__btn--active': active(),
                      'nav-rail__btn--multiline': item.lines.length > 1,
                    }}
                    aria-current={active() ? 'page' : undefined}
                    aria-label={item.lines.join(' ')}
                    onClick={() => select(item.id)}
                  >
                    <span class="nav-rail__metal" aria-hidden="true" />
                    <span class="nav-rail__well">
                      <span class="nav-rail__icon" aria-hidden="true">
                        <NavIcon id={item.id} />
                      </span>
                      <span class="nav-rail__label">
                        <For each={item.lines}>{(line) => <span>{line}</span>}</For>
                      </span>
                    </span>
                    <Show when={active()}>
                      <span class="nav-rail__pip" aria-hidden="true" />
                    </Show>
                  </button>
                </li>
              )
            }}
          </For>
        </ul>
      </nav>
    </Show>
  )
}

function NavIcon(props: { id: NavRailId }): JSX.Element {
  const common = {
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    'stroke-width': 1.75,
    'stroke-linejoin': 'miter' as const,
    'aria-hidden': true as const,
  }

  switch (props.id) {
    case 'parser':
      return (
        <svg {...common}>
          <path d="M12 3 L19 12 L12 21 L5 12 Z" />
          <path d="M9 12 H15" />
        </svg>
      )
    case 'players':
      return (
        <svg {...common}>
          <circle cx="12" cy="8" r="3.25" />
          <path d="M6.5 19.5 C6.5 15.5 9 13.5 12 13.5 C15 13.5 17.5 15.5 17.5 19.5" />
        </svg>
      )
    case 'fantasy':
      return (
        <svg {...common}>
          <path d="M12 3 L18 6.5 V12 C18 16 15.5 19.5 12 21 C8.5 19.5 6 16 6 12 V6.5 Z" />
          <path d="M9.5 12.5 L11.2 14.2 L15 10" />
        </svg>
      )
    case 'leagues':
      return (
        <svg {...common}>
          <path d="M4 7 H20" />
          <path d="M4 12 H20" />
          <path d="M4 17 H14" />
          <path d="M17 15 L20 17.5 L17 20" />
        </svg>
      )
    case 'users':
      return (
        <svg {...common}>
          <circle cx="9" cy="8" r="2.75" />
          <circle cx="16" cy="9" r="2.25" />
          <path d="M4.5 19 C4.5 15.8 6.6 14 9 14 C10.2 14 11.3 14.4 12.1 15.1" />
          <path d="M12.5 19 C12.5 16.2 14.2 14.8 16 14.8 C17.8 14.8 19.5 16.2 19.5 19" />
        </svg>
      )
    case 'titles':
      return (
        <svg {...common}>
          <path d="M12 3 L14.2 8.4 L20 9 L15.6 12.8 L16.9 18.6 L12 15.8 L7.1 18.6 L8.4 12.8 L4 9 L9.8 8.4 Z" />
        </svg>
      )
  }
}
