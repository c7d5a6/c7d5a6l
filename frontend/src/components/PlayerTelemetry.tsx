import { For, Show, createMemo } from 'solid-js'
import { parseRaceId, RACE_META } from '../lib/races'
import {
  displayChangeValue,
  displayValue,
  playerPortraitSrc,
  type PlayerPage,
  type PlayerSync,
} from '../types/tournament'
import { ChannelHead, NestedPlate, RailSection } from './ChannelChrome'
import { Player } from './Player'

export type PlayerTelemetryProps = {
  player: PlayerPage
  sync: PlayerSync
  message: string
}

export function PlayerTelemetry(props: PlayerTelemetryProps) {
  const p = () => props.player
  const sync = () => props.sync
  const raceMeta = () => {
    const id = parseRaceId(p().preferredRace)
    return id ? RACE_META[id] : null
  }

  const dbStatus = createMemo(() => {
    const s = sync()
    if (!s.exists) return 'Not in database'
    if (s.same) return 'In database — matches parse'
    return 'In database — differs from parse'
  })

  return (
    <section class="telemetry channel-stack" aria-label="Player parse result">
      <div class="telemetry__head">
        <span>Player payload</span>
        <span>{props.message}</span>
      </div>

      <ChannelHead tag="Parse" title="Identity" compact />

      <NestedPlate>
        <div class="telemetry__grid">
          <div class="telemetry__row">
            <span class="telemetry__key">DB status</span>
            <span class="telemetry__val">{dbStatus()}</span>
          </div>
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
            <span class="telemetry__key">Portrait</span>
            <span class="telemetry__val">
              <Show
                when={playerPortraitSrc(p())}
                fallback={
                  <Show when={p().portraitUrl} fallback={displayValue(null)}>
                    {(url) => (
                      <span title={url()}>Cached on save</span>
                    )}
                  </Show>
                }
              >
                {(src) => (
                  <img class="telemetry__portrait" src={src()} alt="" />
                )}
              </Show>
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
      </NestedPlate>

      <RailSection label="Registry" title={`IDs (${p().ids.length})`}>
        <Show when={p().ids.length > 0} fallback={<p class="telemetry__val">—</p>}>
          <ul class="telemetry__list">
            <For each={p().ids}>{(id) => <li>{id}</li>}</For>
          </ul>
        </Show>
      </RailSection>

      <Show when={(sync().changes?.length ?? 0) > 0}>
        <RailSection label="Sync" title={`Differences (${sync().changes!.length})`} hazard>
          <div class="telemetry__grid">
            <For each={sync().changes}>
              {(change) => (
                <div class="telemetry__row">
                  <span class="telemetry__key">{change.field}</span>
                  <span class="telemetry__val">
                    {displayChangeValue(change.before)} → {displayChangeValue(change.after)}
                  </span>
                </div>
              )}
            </For>
          </div>
        </RailSection>
      </Show>
    </section>
  )
}
