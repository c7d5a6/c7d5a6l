import { Show, type JSX } from 'solid-js'
import { parseRaceId, RACE_META } from '../lib/races'
import { displayValue } from '../types/tournament'

export type PlayerProps = {
  name: string | null | undefined
  link?: string | null
  race?: string | null
  excluded?: boolean
  loser?: boolean
  class?: string
}

/**
 * Race-aware player chip: icon + name (profile link colored by current race).
 */
export function Player(props: PlayerProps): JSX.Element {
  const race = () => parseRaceId(props.race)
  const meta = () => {
    const id = race()
    return id ? RACE_META[id] : null
  }

  return (
    <span
      classList={{
        player: true,
        'player--excluded': Boolean(props.excluded),
        'player--loser': Boolean(props.loser),
        [meta()?.playerClass ?? '']: Boolean(meta()),
        [props.class ?? '']: Boolean(props.class),
      }}
    >
      <Show when={meta()} fallback={<span class="player__icon-slot" aria-hidden="true" />}>
        {(m) => (
          <img class="player__icon" src={m().icon} alt="" title={m().label} />
        )}
      </Show>

      <Show
        when={props.link}
        fallback={<span class="player__name">{displayValue(props.name)}</span>}
      >
        {(href) => (
          <a class="player__name player__link" href={href()} target="_blank" rel="noreferrer">
            {displayValue(props.name)}
          </a>
        )}
      </Show>

      <Show when={props.excluded}>
        <span class="player__tag">excluded</span>
      </Show>
    </span>
  )
}
