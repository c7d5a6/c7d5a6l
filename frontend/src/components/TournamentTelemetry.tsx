import { For, Match, Show, Switch, createMemo, createSignal } from 'solid-js'
import { RACE_META } from '../lib/races'
import {
  displayChangeValue,
  displayValue,
  type TournamentGroup,
  type TournamentPage,
  type TournamentSync,
} from '../types/tournament'
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

  const groupsByPhase = createMemo(() => {
    const map = new Map<string, TournamentGroup[]>()
    for (const g of groups()) {
      const phase = g.phase || '—'
      const list = map.get(phase) ?? []
      list.push(g)
      map.set(phase, list)
    }
    return [...map.entries()]
  })

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
              <For each={groupsByPhase()}>
                {([phase, phaseGroups]) => (
                  <div class="telemetry__block">
                    <div class="telemetry__block-head">{phase}</div>
                    <For each={phaseGroups}>
                      {(g) => (
                        <div class="telemetry__group">
                          <div class="telemetry__group-name">{g.name}</div>
                          <ul class="telemetry__list telemetry__list--chips">
                            <For each={g.players}>
                              {(p) => (
                                <li>
                                  <Player
                                    name={p.name}
                                    link={p.link}
                                    race={p.race}
                                    excluded={p.excluded}
                                  />
                                </li>
                              )}
                            </For>
                          </ul>
                        </div>
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
              <ul class="telemetry__matches">
                <For each={t().results}>
                  {(m) => (
                    <li classList={{ telemetry__match: true, 'telemetry__match--pending': !m.played }}>
                      <span class="telemetry__match-order">#{m.order}</span>
                      <div class="telemetry__match-body">
                        <div class="telemetry__match-meta">
                          <span class="telemetry__match-stage">{displayValue(m.stage)}</span>
                          <span class="telemetry__match-time">{displayValue(m.dateTime)}</span>
                        </div>
                        <div class="telemetry__match-line">
                          <div class="telemetry__match-side telemetry__match-side--a">
                            <Show when={m.participantA} fallback={<span class="telemetry__val">—</span>}>
                              {(p) => (
                                <Player
                                  name={p().name}
                                  link={p().link}
                                  race={p().race}
                                  excluded={p().excluded}
                                  loser={
                                    m.played &&
                                    m.scoreA != null &&
                                    m.scoreB != null &&
                                    m.scoreA < m.scoreB
                                  }
                                />
                              )}
                            </Show>
                          </div>
                          <span class="telemetry__match-score">
                            <Show when={m.played} fallback={<span class="telemetry__match-vs">vs</span>}>
                              {displayValue(m.scoreA)}:{displayValue(m.scoreB)}
                            </Show>
                          </span>
                          <div class="telemetry__match-side telemetry__match-side--b">
                            <Show when={m.participantB} fallback={<span class="telemetry__val">—</span>}>
                              {(p) => (
                                <Player
                                  name={p().name}
                                  link={p().link}
                                  race={p().race}
                                  excluded={p().excluded}
                                  loser={
                                    m.played &&
                                    m.scoreA != null &&
                                    m.scoreB != null &&
                                    m.scoreB < m.scoreA
                                  }
                                />
                              )}
                            </Show>
                          </div>
                        </div>
                      </div>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </div>
        </Match>
      </Switch>
    </section>
  )
}
