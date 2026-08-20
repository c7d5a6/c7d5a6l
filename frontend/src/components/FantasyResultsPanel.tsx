import { For, Match, Show, Switch, createMemo, type JSX } from 'solid-js'
import { GroupCard } from './GroupCard'
import { isPlayoffsPhase, partitionGroups, type GroupWithResults } from '../lib/matchBoard'
import type { FantasyGroup, FantasyMatchBoard } from '../types/fantasy'

export type FantasyResultsPanelProps = {
  board: FantasyMatchBoard | undefined
  loading: boolean
  error: unknown
}

/** Upcoming / completed tournament groups with match scores. */
export function FantasyResultsPanel(props: FantasyResultsPanelProps): JSX.Element {
  const parts = createMemo(() => {
    const b = props.board
    if (!b) return { upcoming: [] as GroupWithResults<FantasyGroup>[], completed: [] as GroupWithResults<FantasyGroup>[] }
    return partitionGroups(b.groups, b.results, b.today)
  })

  return (
    <div class="telemetry__panel fantasy-results" role="tabpanel">
      <Switch>
        <Match when={props.loading}>
          <p class="status status--idle">Loading match board…</p>
        </Match>
        <Match when={props.error}>
          <p class="status status--error">
            {(props.error as Error)?.message ?? 'Failed to load match board'}
          </p>
        </Match>
        <Match when={!props.board || (props.board.groups.length === 0 && props.board.results.length === 0)}>
          <p class="status status--idle">No groups or results yet</p>
        </Match>
        <Match when={true}>
          <ResultsSection title="Upcoming" entries={parts().upcoming} />
          <ResultsSection title="Completed" entries={parts().completed} />
        </Match>
      </Switch>
    </div>
  )
}

function ResultsSection(props: {
  title: string
  entries: GroupWithResults<FantasyGroup>[]
}): JSX.Element {
  const phases = createMemo(() => {
    const map = new Map<string, GroupWithResults<FantasyGroup>[]>()
    for (const e of props.entries) {
      const phase = e.group.phase || '—'
      const list = map.get(phase) ?? []
      list.push(e)
      map.set(phase, list)
    }
    return [...map.entries()]
  })

  return (
    <Show when={props.entries.length > 0}>
      <section class="fantasy-results__section">
        <header class="fantasy-results__section-head">
          <span class="fantasy-results__section-tag">Match board</span>
          <h2 class="fantasy-results__section-title">{props.title}</h2>
        </header>

        <For each={phases()}>
          {([phase, entries]) => (
            <div
              classList={{
                'fantasy-results__phase': true,
                'fantasy-results__phase--playoffs': isPlayoffsPhase(phase),
              }}
            >
              <header class="fantasy-results__phase-head">
                <span class="fantasy-results__phase-rail" aria-hidden="true" />
                <div class="fantasy-results__phase-copy">
                  <span class="fantasy-results__phase-label">Phase</span>
                  <h3 class="fantasy-results__phase-title">{phase}</h3>
                </div>
              </header>

              <div
                classList={{
                  'fantasy-results__grid': true,
                  'fantasy-results__grid--bracket': isPlayoffsPhase(phase),
                }}
              >
                <For each={entries}>
                  {(e) => (
                    <GroupCard
                      name={e.group.name}
                      playoff={isPlayoffsPhase(e.group.phase)}
                      players={e.group.players}
                      results={e.results}
                      dense
                    />
                  )}
                </For>
              </div>
            </div>
          )}
        </For>
      </section>
    </Show>
  )
}
