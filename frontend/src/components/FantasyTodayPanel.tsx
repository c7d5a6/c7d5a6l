import { For, Show, createMemo, type JSX } from 'solid-js'
import { MatchRow } from './GroupCard'
import { Player } from './Player'
import { participantLinksInResults, pickDayMatches } from '../lib/matchBoard'
import type { FantasyMatchBoard, FantasyTeamRow } from '../types/fantasy'

export type FantasyTodayPanelProps = {
  board: FantasyMatchBoard
  teams: FantasyTeamRow[]
}

/** Today (or next match day) + operators who roster those players. */
export function FantasyTodayPanel(props: FantasyTodayPanelProps): JSX.Element {
  const day = createMemo(() => pickDayMatches(props.board.results, props.board.today))
  const links = createMemo(() => participantLinksInResults(day().matches))
  const operators = createMemo(() => {
    const want = links()
    if (want.size === 0) return []
    return props.teams
      .map((team) => {
        const members = team.members.filter((m) => {
          const link = m.link?.trim().toLowerCase()
          return link && want.has(link)
        })
        return { team, members }
      })
      .filter((x) => x.members.length > 0)
  })

  return (
    <div class="fantasy-today">
      <header class="fantasy-groups__head">
        <p class="brand__eyebrow">Match day</p>
        <h2 class="fantasy-groups__title">{day().label}</h2>
      </header>
      <hr class="rule" />

      <Show
        when={day().matches.length > 0}
        fallback={<p class="status status--idle">No scheduled matches</p>}
      >
        <div class="fantasy-today__matches">
          <For each={day().matches}>{(m) => <MatchRow result={m} compact />}</For>
        </div>
      </Show>

      <Show when={operators().length > 0}>
        <div class="telemetry__block-head" style={{ 'margin-top': '0.85rem' }}>
          Operators
        </div>
        <ul class="fantasy-today__ops">
          <For each={operators()}>
            {(row) => (
              <li class="fantasy-today__op">
                <span class="fantasy-today__alias">{row.team.userAlias}</span>
                <ul class="telemetry__list telemetry__list--chips">
                  <For each={row.members}>
                    {(m) => (
                      <li>
                        <Player name={m.name} link={m.link} race={m.race} />
                      </li>
                    )}
                  </For>
                </ul>
              </li>
            )}
          </For>
        </ul>
      </Show>
    </div>
  )
}
