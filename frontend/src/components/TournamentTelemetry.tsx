import { For, Show } from 'solid-js'
import { RACE_META } from '../lib/races'
import { displayValue, type TournamentPage } from '../types/tournament'

const RACE_STATS = ['protoss', 'terran', 'zerg'] as const

export function TournamentTelemetry(props: { tournament: TournamentPage; message: string }) {
  const t = () => props.tournament
  const counts = () => t().playerCounts

  return (
    <section class="telemetry" aria-label="Tournament parse result">
      <div class="telemetry__head">
        <span>Tournament payload</span>
        <span>{props.message}</span>
      </div>

      <div class="telemetry__grid">
        <div class="telemetry__row">
          <span class="telemetry__key">Link</span>
          <a class="telemetry__val telemetry__link" href={t().link} target="_blank" rel="noreferrer">
            {t().link}
          </a>
        </div>
        <div class="telemetry__row">
          <span class="telemetry__key">Name</span>
          <span class="telemetry__val">{displayValue(t().name)}</span>
        </div>
        <div class="telemetry__row">
          <span class="telemetry__key">Start date</span>
          <span class="telemetry__val">{displayValue(t().startDate)}</span>
        </div>
        <div class="telemetry__row">
          <span class="telemetry__key">End date</span>
          <span class="telemetry__val">{displayValue(t().endDate)}</span>
        </div>
        <div class="telemetry__row">
          <span class="telemetry__key">Liquipedia tier</span>
          <span class="telemetry__val">{displayValue(t().liquipediaTier)}</span>
        </div>
        <div class="telemetry__row">
          <span class="telemetry__key">Finished</span>
          <span class="telemetry__val">{displayValue(t().finished)}</span>
        </div>
      </div>

      <div class="telemetry__block">
        <div class="telemetry__block-head">Player counts</div>
        <Show when={counts()} fallback={<p class="telemetry__val">—</p>}>
          {(c) => (
            <div class="telemetry__stats">
              <div class="telemetry__stat">
                <span class="telemetry__stat-label">Total</span>
                <span class="telemetry__stat-val">{displayValue(c().total)}</span>
              </div>
              <For each={[...RACE_STATS]}>
                {(race) => {
                  const meta = RACE_META[race]
                  return (
                    <div class={`telemetry__stat ${meta.statClass}`}>
                      <span class="telemetry__stat-label">
                        <img class="telemetry__stat-icon" src={meta.icon} alt="" />
                        {meta.label}
                      </span>
                      <span class="telemetry__stat-val">{displayValue(c()[race])}</span>
                    </div>
                  )
                }}
              </For>
            </div>
          )}
        </Show>
      </div>

      <div class="telemetry__block">
        <div class="telemetry__block-head">
          Participants <span>({t().participants.length})</span>
        </div>
        <Show
          when={t().participants.length > 0}
          fallback={<p class="telemetry__val">No participants parsed yet</p>}
        >
          <ul class="telemetry__list">
            <For each={t().participants}>
              {(p) => <li class="telemetry__val">{displayValue(p.name)}</li>}
            </For>
          </ul>
        </Show>
      </div>

      <div class="telemetry__block">
        <div class="telemetry__block-head">
          Results <span>({t().results.length})</span>
        </div>
        <p class="telemetry__val">Results model standing by — empty</p>
      </div>
    </section>
  )
}
