import { For, Show, createMemo, type JSX } from 'solid-js'
import { ChannelHead, NestedPlate, RailSection } from './ChannelChrome'
import { MatchRow } from './GroupCard'
import { GroupWinnerMark } from './GroupWinnerMark'
import { Player } from './Player'
import {
  groupWinnerLinksForDay,
  groupWinnersForDay,
  nextMatchHint,
  normPlayerLink,
  participantLinksInResults,
  pickDayMatches,
} from '../lib/matchBoard'
import type { FantasyMatchBoard, FantasyTeamRow } from '../types/fantasy'

export type FantasyTodayPanelProps = {
  board: FantasyMatchBoard
  teams: FantasyTeamRow[]
  /** Fantasy defeated links (lowercase). */
  defeatedLinks?: Set<string>
}

/** Today (or next match day) + operators who roster those players. */
export function FantasyTodayPanel(props: FantasyTodayPanelProps): JSX.Element {
  const day = createMemo(() => pickDayMatches(props.board.results, props.board.today))
  const links = createMemo(() => participantLinksInResults(day().matches))
  const dayGroupWinners = createMemo(() =>
    groupWinnerLinksForDay(props.board.groups, day().matches),
  )
  const dayWinnerPlayers = createMemo(() =>
    groupWinnersForDay(props.board.groups, day().matches),
  )
  const nextHint = createMemo(() => nextMatchHint(props.board.results, props.board.today))
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

  function isDefeated(link: string | null | undefined): boolean {
    const n = normPlayerLink(link)
    return Boolean(n && props.defeatedLinks?.has(n))
  }

  function isDayGroupWinner(link: string | null | undefined): boolean {
    const n = normPlayerLink(link)
    return Boolean(n && dayGroupWinners().has(n))
  }

  return (
    <div class="fantasy-today channel-stack">
      <ChannelHead
        tag="Match day"
        title={day().label}
        compact
        actions={
          <Show when={nextHint()}>
            {(h) => <span class="fantasy-today__next">{h()}</span>}
          </Show>
        }
      />

      <Show
        when={day().matches.length > 0}
        fallback={<p class="status status--idle">No scheduled matches</p>}
      >
        <NestedPlate class="nested-plate--flush">
          <div class="fantasy-today__matches group-card__matches">
            <For each={day().matches}>
              {(m) => (
                <MatchRow result={m} compact defeatedLinks={props.defeatedLinks} />
              )}
            </For>
          </div>
        </NestedPlate>
      </Show>

      <Show when={dayWinnerPlayers().length > 0}>
        <RailSection label="Standings" title="Group winners">
          <ul class="telemetry__list telemetry__list--chips fantasy-today__winners">
            <For each={dayWinnerPlayers()}>
              {(p) => (
                <li
                  class="group-card__player group-card__player--winner"
                  title={`${p.phase} · ${p.groupName}`}
                >
                  <Player
                    name={p.name}
                    link={p.link}
                    race={p.race}
                    loser={isDefeated(p.link)}
                  />
                </li>
              )}
            </For>
          </ul>
        </RailSection>
      </Show>

      <Show when={operators().length > 0}>
        <RailSection label="Channel" title="Operators">
          <ul class="fantasy-today__ops">
            <For each={operators()}>
              {(row) => (
                <li class="fantasy-today__op">
                  <span class="fantasy-today__alias">{row.team.userAlias}</span>
                  <ul class="telemetry__list telemetry__list--chips">
                    <For each={row.members}>
                      {(m) => (
                        <li class="fantasy-today__op-player">
                          <Show when={isDayGroupWinner(m.link)}>
                            <GroupWinnerMark />
                          </Show>
                          <Player
                            name={m.name}
                            link={m.link}
                            race={m.race}
                            loser={isDefeated(m.link)}
                          />
                        </li>
                      )}
                    </For>
                  </ul>
                </li>
              )}
            </For>
          </ul>
        </RailSection>
      </Show>
    </div>
  )
}
