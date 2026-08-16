import type { JSX } from 'solid-js'
import { Show } from 'solid-js'
import { ChampionMark } from './ChampionMark'
import { Player } from './Player'
import { parseRaceId } from '../lib/races'

export type RosterPlayerChipProps = {
  name: string | null | undefined
  link?: string | null
  race?: string | null
  cost: number
  points: number
  defeated?: boolean
  isWinner?: boolean
  /** Fantasy player id — used to peer-highlight the same pick across teams. */
  playerId?: number
  highlighted?: boolean
  onHighlight?: (playerId: number | null) => void
}

/**
 * Compact roster plate for team cards — edit-pick silhouette, race chrome,
 * cost/pts readouts; defeated reads disabled; champion = pulsing star + rim.
 */
export function RosterPlayerChip(props: RosterPlayerChipProps): JSX.Element {
  const race = () => parseRaceId(props.race)
  const canPeer = () => props.playerId != null && props.onHighlight != null

  return (
    <span
      classList={{
        'roster-chip': true,
        'roster-chip--defeated': Boolean(props.defeated),
        'roster-chip--winner': Boolean(props.isWinner),
        'roster-chip--lit': Boolean(props.highlighted),
        'roster-chip--peerable': canPeer(),
        [`roster-chip--${race() ?? 'unknown'}`]: true,
      }}
      onPointerEnter={() => {
        if (props.playerId != null) props.onHighlight?.(props.playerId)
      }}
      onPointerLeave={() => props.onHighlight?.(null)}
    >
      <Show when={props.isWinner}>
        <ChampionMark />
      </Show>
      <Player name={props.name} link={props.link} race={props.race} loser={props.defeated && !props.isWinner} />
      <span class="roster-chip__nums" aria-label={`cost ${props.cost}, points ${props.points}`}>
        <span class="roster-chip__cost" title="Cost">
          {props.cost}
        </span>
        <span class="roster-chip__pts" title="Points">
          {props.points}
        </span>
      </span>
    </span>
  )
}
