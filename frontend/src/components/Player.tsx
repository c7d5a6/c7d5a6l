import { Portal } from 'solid-js/web'
import { For, Show, createEffect, createSignal, onCleanup, type JSX } from 'solid-js'
import {
  loadPlayerInfo,
  loadPlayerPortrait,
  peekPlayerInfo,
  peekPlayerPortrait,
  type PlayerLookupResult,
} from '../lib/playerHoverCache'
import { parseRaceId, RACE_META } from '../lib/races'
import { displayValue, type PlayerPage } from '../types/tournament'

const SHOW_DELAY_MS = 360
const HIDE_DELAY_MS = 140
const CARD_W = 280
const CARD_GAP = 10

type HintState = {
  link: string
  name?: string | null
  race?: string | null
  hasPortrait?: boolean
  anchor: DOMRect
}

/**
 * SC-styled hover dossier: shell shows after a short delay; info + portrait
 * fill in from separate cached requests.
 */
function PlayerHintCard(props: HintState): JSX.Element {
  const [lookup, setLookup] = createSignal<PlayerLookupResult | 'loading'>(
    peekPlayerInfo(props.link) ?? 'loading',
  )
  const [portraitSrc, setPortraitSrc] = createSignal<string | null>(
    peekPlayerPortrait(props.link) ?? null,
  )

  const pos = () => placeCard(props.anchor)

  createEffect(() => {
    const link = props.link
    let cancelled = false

    setLookup(peekPlayerInfo(link) ?? 'loading')
    setPortraitSrc(peekPlayerPortrait(link) ?? null)

    void loadPlayerInfo(link).then((result) => {
      if (cancelled) return
      setLookup(result)
      if (result.status === 'ok' && result.player.hasPortrait) {
        void loadPlayerPortrait(link, true).then((src) => {
          if (!cancelled && src) setPortraitSrc(src)
        })
      }
    })

    if (props.hasPortrait) {
      void loadPlayerPortrait(link, true).then((src) => {
        if (!cancelled && src) setPortraitSrc(src)
      })
    }

    onCleanup(() => {
      cancelled = true
    })
  })

  const displayName = () => {
    const L = lookup()
    if (L !== 'loading' && L.status === 'ok') return displayValue(L.player.name)
    return displayValue(props.name)
  }

  const displayRace = () => {
    const L = lookup()
    if (L !== 'loading' && L.status === 'ok') return L.player.preferredRace
    return props.race ?? null
  }

  const raceMeta = () => {
    const id = parseRaceId(displayRace())
    return id ? RACE_META[id] : null
  }

  const okPlayer = (): PlayerPage | null => {
    const L = lookup()
    return L !== 'loading' && L.status === 'ok' ? L.player : null
  }

  const statusLine = () => {
    const L = lookup()
    if (L === 'loading') return 'Locking dossier…'
    if (L.status === 'missing') return 'Not in local database'
    if (L.status === 'error') return L.message
    return null
  }

  return (
    <Portal>
      <div
        class="player-hint"
        style={{
          left: `${pos().left}px`,
          top: `${pos().top}px`,
        }}
        role="tooltip"
      >
        <div class="player-hint__well">
          <div class="player-hint__portrait">
            <Show
              when={portraitSrc()}
              fallback={<div class="player-hint__portrait-empty" aria-hidden="true" />}
            >
              {(src) => <img src={src()} alt="" />}
            </Show>
          </div>
          <div class="player-hint__body">
            <div class="player-hint__name-row">
              <Show when={raceMeta()}>
                {(m) => <img class="player-hint__race" src={m().icon} alt="" title={m().label} />}
              </Show>
              <span class="player-hint__name">{displayName()}</span>
            </div>
            <Show when={okPlayer()}>
              {(p) => (
                <dl class="player-hint__meta">
                  <div>
                    <dt>Real</dt>
                    <dd>{displayValue(p().realName)}</dd>
                  </div>
                  <div>
                    <dt>Race</dt>
                    <dd>{displayValue(p().preferredRace)}</dd>
                  </div>
                  <Show when={(p().raceElos ?? []).length > 0}>
                    <div class="player-hint__elos">
                      <dt>Elo</dt>
                      <dd>
                        <For each={p().raceElos ?? []}>
                          {(re) => {
                            const meta = () => {
                              const id = parseRaceId(re.race)
                              return id ? RACE_META[id] : null
                            }
                            return (
                              <span class="player-hint__elo">
                                <Show when={meta()}>
                                  {(m) => <img src={m().icon} alt="" title={m().label} />}
                                </Show>
                                {re.elo.toFixed(0)}
                              </span>
                            )
                          }}
                        </For>
                      </dd>
                    </div>
                  </Show>
                  <Show when={p().ids.length > 0}>
                    <div class="player-hint__ids">
                      <dt>IDs</dt>
                      <dd>
                        <For each={p().ids}>
                          {(id) => <span class="player-hint__id">{id}</span>}
                        </For>
                      </dd>
                    </div>
                  </Show>
                </dl>
              )}
            </Show>
            <Show when={statusLine()}>
              {(line) => <p class="player-hint__status">{line()}</p>}
            </Show>
          </div>
        </div>
      </div>
    </Portal>
  )
}

function placeCard(anchor: DOMRect): { left: number; top: number } {
  const vw = window.innerWidth
  const vh = window.innerHeight
  let left = anchor.left
  let top = anchor.bottom + CARD_GAP

  if (left + CARD_W > vw - 8) left = Math.max(8, vw - CARD_W - 8)
  if (left < 8) left = 8

  const estimatedH = 130
  if (top + estimatedH > vh - 8) {
    top = Math.max(8, anchor.top - estimatedH - CARD_GAP)
  }
  return { left, top }
}

export type PlayerProps = {
  name: string | null | undefined
  link?: string | null
  race?: string | null
  excluded?: boolean
  loser?: boolean
  /** Match winner — bold name (not fantasy champion). */
  winner?: boolean
  /** When known (e.g. roster), portrait fetch can start with the shell. */
  hasPortrait?: boolean
  class?: string
}

/**
 * Race-aware player chip: icon + name (profile link colored by current race).
 * Linked names open a delayed hover dossier.
 */
export function Player(props: PlayerProps): JSX.Element {
  const race = () => parseRaceId(props.race)
  const meta = () => {
    const id = race()
    return id ? RACE_META[id] : null
  }

  const [hint, setHint] = createSignal<HintState | null>(null)
  let showTimer = 0
  let hideTimer = 0

  onCleanup(() => {
    window.clearTimeout(showTimer)
    window.clearTimeout(hideTimer)
  })

  function openHint(el: HTMLElement) {
    const link = props.link
    if (!link || link.startsWith('local://')) return
    window.clearTimeout(hideTimer)
    window.clearTimeout(showTimer)
    showTimer = window.setTimeout(() => {
      setHint({
        link,
        name: props.name,
        race: props.race,
        hasPortrait: props.hasPortrait,
        anchor: el.getBoundingClientRect(),
      })
    }, SHOW_DELAY_MS)
  }

  function scheduleClose() {
    window.clearTimeout(showTimer)
    window.clearTimeout(hideTimer)
    hideTimer = window.setTimeout(() => setHint(null), HIDE_DELAY_MS)
  }

  return (
    <span
      classList={{
        player: true,
        'player--excluded': Boolean(props.excluded),
        'player--loser': Boolean(props.loser),
        'player--winner': Boolean(props.winner) && !props.loser,
        [meta()?.playerClass ?? '']: Boolean(meta()),
        [props.class ?? '']: Boolean(props.class),
      }}
    >
      <Show when={meta()} fallback={<span class="player__icon-slot" aria-hidden="true" />}>
        {(m) => <img class="player__icon" src={m().icon} alt="" title={m().label} />}
      </Show>

      <Show
        when={props.link && !props.link.startsWith('local://')}
        fallback={<span class="player__name">{displayValue(props.name)}</span>}
      >
        {(href) => (
          <a
            class="player__name player__link"
            href={href()}
            target="_blank"
            rel="noreferrer"
            onMouseEnter={(e) => openHint(e.currentTarget)}
            onMouseLeave={scheduleClose}
            onFocus={(e) => openHint(e.currentTarget)}
            onBlur={scheduleClose}
          >
            {displayValue(props.name)}
          </a>
        )}
      </Show>

      <Show when={props.excluded}>
        <span class="player__tag">excluded</span>
      </Show>

      <Show when={hint()}>
        {(h) => (
          <PlayerHintCard
            link={h().link}
            name={h().name}
            race={h().race}
            hasPortrait={h().hasPortrait}
            anchor={h().anchor}
          />
        )}
      </Show>
    </span>
  )
}
