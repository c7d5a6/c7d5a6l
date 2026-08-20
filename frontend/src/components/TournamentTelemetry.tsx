import { For, Match, Show, Switch, createMemo, createSignal } from 'solid-js'
import { RACE_META } from '../lib/races'
import { groupsByPhase } from '../lib/groupsByPhase'
import { resultsForGroup } from '../lib/matchBoard'
import {
  displayChangeValue,
  displayValue,
  type TournamentPage,
  type TournamentSync,
} from '../types/tournament'
import { GroupCard, MatchRow } from './GroupCard'
import { Player } from './Player'

const RACE_STATS = ['protoss', 'terran', 'zerg'] as const
type TabId = 'overview' | 'players' | 'groups' | 'results'

export function TournamentTelemetry(props: {
  tournament: TournamentPage
  sync: TournamentSync
  message: string
}) {
  const [tab, setTab] = createSignal<TabId>('overview')
  const t = () => props.tournament
  const sync = () => props.sync
  const counts = () => t().playerCounts
  const groups = () => t().groups ?? []

  const byPhase = createMemo(() => groupsByPhase(groups()))

  const dbStatus = createMemo(() => {
    const s = sync()
    if (!s.exists) return 'Not in database'
    if (s.same) return 'In database — matches parse'
    return 'In database — differs from parse'
  })

  const toImport = createMemo(() => (sync().players ?? []).filter((p) => p.willImport))
  const changes = createMemo(() => sync().changes ?? [])

  return (
    <section class="telemetry" aria-label="Tournament parse result">
      <div class="telemetry__head">
        <span>Tournament payload</span>
        <span>{props.message}</span>
      </div>

      <nav class="console__tabs" role="tablist" aria-label="Tournament sections">
        <button
          type="button"
          role="tab"
          classList={{ tab: true, 'tab--active': tab() === 'overview' }}
          aria-selected={tab() === 'overview'}
          onClick={() => setTab('overview')}
        >
          Overview
        </button>
        <button
          type="button"
          role="tab"
          classList={{ tab: true, 'tab--active': tab() === 'players' }}
          aria-selected={tab() === 'players'}
          onClick={() => setTab('players')}
        >
          Players
        </button>
        <button
          type="button"
          role="tab"
          classList={{ tab: true, 'tab--active': tab() === 'groups' }}
          aria-selected={tab() === 'groups'}
          onClick={() => setTab('groups')}
        >
          Groups
        </button>
        <button
          type="button"
          role="tab"
          classList={{ tab: true, 'tab--active': tab() === 'results' }}
          aria-selected={tab() === 'results'}
          onClick={() => setTab('results')}
        >
          Results
        </button>
      </nav>

      <Switch>
        <Match when={tab() === 'overview'}>
          <div class="telemetry__panel" role="tabpanel">
            <div class="telemetry__grid">
              <div class="telemetry__row">
                <span class="telemetry__key">DB status</span>
                <span class="telemetry__val">{dbStatus()}</span>
              </div>
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

            <Show when={changes().length > 0}>
              <div class="telemetry__block">
                <div class="telemetry__block-head">
                  Differences <span>({changes().length})</span>
                </div>
                <div class="telemetry__grid">
                  <For each={changes()}>
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
              </div>
            </Show>

            <Show when={toImport().length > 0}>
              <div class="telemetry__block">
                <div class="telemetry__block-head">
                  Players to import <span>({toImport().length})</span>
                </div>
                <ul class="telemetry__list">
                  <For each={toImport()}>
                    {(p) => (
                      <li>
                        <Player name={p.name} link={p.link} race={p.race} excluded={p.excluded} />
                      </li>
                    )}
                  </For>
                </ul>
              </div>
            </Show>
          </div>
        </Match>

        <Match when={tab() === 'players'}>
          <div class="telemetry__panel" role="tabpanel">
            <div class="telemetry__block-head">
              Participants <span>({t().participants.length})</span>
            </div>
            <Show
              when={t().participants.length > 0}
              fallback={<p class="telemetry__val">No participants parsed yet</p>}
            >
              <ul class="telemetry__list">
                <For each={t().participants}>
                  {(p) => (
                    <li>
                      <Player name={p.name} link={p.link} race={p.race} excluded={p.excluded} />
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </div>
        </Match>

        <Match when={tab() === 'groups'}>
          <div class="telemetry__panel" role="tabpanel">
            <div class="telemetry__block-head">
              Groups <span>({groups().length})</span>
            </div>
            <Show
              when={groups().length > 0}
              fallback={<p class="telemetry__val">No groups parsed yet</p>}
            >
              <For each={byPhase()}>
                {([phase, phaseGroups]) => (
                  <div class="telemetry__block">
                    <div class="telemetry__block-head">{phase}</div>
                    <For each={phaseGroups}>
                      {(g) => (
                        <GroupCard
                          name={g.name}
                          players={g.players}
                          results={resultsForGroup(t().results ?? [], g.id, g.phase, g.name)}
                        />
                      )}
                    </For>
                  </div>
                )}
              </For>
            </Show>
          </div>
        </Match>

        <Match when={tab() === 'results'}>
          <div class="telemetry__panel" role="tabpanel">
            <div class="telemetry__block-head">
              Results <span>({t().results.length})</span>
            </div>
            <Show
              when={t().results.length > 0}
              fallback={<p class="telemetry__val">No results parsed yet</p>}
            >
              <div class="group-card__matches telemetry__match-list">
                <For each={t().results}>{(m) => <MatchRow result={m} />}</For>
              </div>
            </Show>
          </div>
        </Match>
      </Switch>
    </section>
  )
}
