import { For, Show } from 'solid-js'
import { parseRaceId, RACE_META } from '../lib/races'
import { displayValue, type PlayerPage } from '../types/tournament'
import { Player } from './Player'

export function PlayerTelemetry(props: { player: PlayerPage; message: string }) {
  const p = () => props.player
  const raceMeta = () => {
    const id = parseRaceId(p().preferredRace)
    return id ? RACE_META[id] : null
  }

  return (
    <section class="telemetry" aria-label="Player parse result">
      <div class="telemetry__head">
        <span>Player payload</span>
        <span>{props.message}</span>
      </div>

      <div class="telemetry__grid">
        <div class="telemetry__row">
          <span class="telemetry__key">Link</span>
          <a class="telemetry__val telemetry__link" href={p().link} target="_blank" rel="noreferrer">
            {p().link}
          </a>
        </div>
        <div class="telemetry__row">
          <span class="telemetry__key">Name</span>
          <span class="telemetry__val">
            <Player name={p().name} link={p().link} race={p().preferredRace} />
          </span>
        </div>
        <div class="telemetry__row">
          <span class="telemetry__key">Real name</span>
          <span class="telemetry__val">{displayValue(p().realName)}</span>
        </div>
        <div class="telemetry__row">
          <span class="telemetry__key">Preferred race</span>
          <span class="telemetry__val">
            <Show when={raceMeta()} fallback={displayValue(p().preferredRace)}>
              {(meta) => (
                <span class={`telemetry__stat ${meta().statClass}`}>
                  <span class="telemetry__stat-label">
                    <img class="telemetry__stat-icon" src={meta().icon} alt="" />
                    {meta().label}
                  </span>
                </span>
              )}
            </Show>
          </span>
        </div>
      </div>

      <div class="telemetry__block">
        <div class="telemetry__block-head">
          IDs <span>({p().ids.length})</span>
        </div>
        <Show
          when={p().ids.length > 0}
          fallback={<p class="telemetry__val">—</p>}
        >
          <ul class="telemetry__list">
            <For each={p().ids}>{(id) => <li>{id}</li>}</For>
          </ul>
        </Show>
      </div>
    </section>
  )
}
