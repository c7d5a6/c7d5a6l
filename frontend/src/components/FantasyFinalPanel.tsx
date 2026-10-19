import { For, Show, createMemo, type JSX } from 'solid-js'
import { ChannelHead, RailSection } from './ChannelChrome'
import { ChampionMark } from './ChampionMark'
import { Player } from './Player'
import { RosterPlayerChip } from './RosterPlayerChip'
import { TeamScoreMeta } from './FantasyScoreReadout'
import { UserTitles } from './UserTitles'
import { titleImageSrc, type UserTitle } from '../types/user'
import type { FantasyPlayerRow, FantasyTeamRow } from '../types/fantasy'

export type FantasyFinalPanelProps = {
  leagueId: number
  teams: FantasyTeamRow[]
  players: FantasyPlayerRow[]
}

/** Finished-league outcome: rank-1 fantasy teams and the tournament champion. */
export function FantasyFinalPanel(props: FantasyFinalPanelProps): JSX.Element {
  const winners = createMemo(() => props.teams.filter((team) => team.rank === 1))
  const champion = createMemo(() => props.players.find((player) => player.isWinner) ?? null)

  return (
    <div class="fantasy-today channel-stack">
      <ChannelHead tag="Final" title="Champions" compact />

      <RailSection label="Fantasy" title="Winning team">
        <Show
          when={winners().length > 0}
          fallback={<p class="status status--idle">No fantasy teams</p>}
        >
          <ul class="fantasy-today__ops">
            <For each={winners()}>
              {(team) => {
                const leagueTitles = () =>
                  (team.titles ?? []).filter((title) => title.fantasyLeagueId === props.leagueId)
                return (
                  <li class="fantasy-today__op">
                    <div class="fantasy-final__identity">
                      <span class="fantasy-today__alias">{team.userAlias}</span>
                      <UserTitles titles={team.titles} />
                    </div>
                    <For each={leagueTitles()}>
                      {(title) => <LeagueTitle title={title} />}
                    </For>
                    <TeamScoreMeta points={team.points} cost={team.cost} />
                    <ul class="fantasy-team__list">
                      <For each={team.members}>
                        {(member) => (
                          <li class="fantasy-team__slot">
                            <RosterPlayerChip
                              name={member.name}
                              link={member.link}
                              race={member.race}
                              cost={member.cost}
                              points={member.pointsEarned}
                              defeated={member.defeated}
                              isWinner={member.isWinner}
                            />
                          </li>
                        )}
                      </For>
                    </ul>
                  </li>
                )
              }}
            </For>
          </ul>
        </Show>
      </RailSection>

      <RailSection label="Tournament" title="Champion">
        <Show
          when={champion()}
          fallback={<p class="status status--idle">Champion not marked</p>}
        >
          {(player) => (
            <div class="fantasy-final__champion">
              <ChampionMark />
              <Player name={player().name} link={player().link} race={player().race} />
              <span class="fantasy-final__champion-pts">{player().pointsEarned} PTS</span>
            </div>
          )}
        </Show>
      </RailSection>
    </div>
  )
}

function LeagueTitle(props: { title: UserTitle }): JSX.Element {
  const title = () => props.title
  return (
    <div class="fantasy-final__league-title">
      <Show when={title().hasImage}>
        <span class="user-title user-title--feature" title={title().name} aria-hidden="true">
          <span class="user-title__hex">
            <img class="user-title__photo" src={titleImageSrc(title().id)} alt="" />
            <svg class="user-title__hex-stroke" viewBox="0 0 24 24">
              <path d="M12 1.6 L21.4 7 V17 L12 22.4 L2.6 17 V7 Z" />
            </svg>
          </span>
        </span>
      </Show>
      <span class="fantasy-final__league-title-name">{title().name}</span>
    </div>
  )
}
