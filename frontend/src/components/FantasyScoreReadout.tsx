import type { JSX } from 'solid-js'
import { Show } from 'solid-js'

/** Team header readout — cost quiet metal; points phosphor on the right. */
export function TeamScoreMeta(props: {
  points: number
  cost: number
  players?: number
}): JSX.Element {
  return (
    <span class="fantasy-team__meta">
      <Show when={props.players != null}>
        <span class="fantasy-readout fantasy-readout--dim">
          <span class="fantasy-readout__val">{props.players}</span>
          <span class="fantasy-readout__unit">PLY</span>
        </span>
      </Show>
      <span class="fantasy-readout fantasy-readout--cost" title="Roster cost">
        <span class="fantasy-readout__val">{props.cost}</span>
        <span class="fantasy-readout__unit">COST</span>
      </span>
      <span class="fantasy-readout fantasy-readout--pts" title="Points earned">
        <span class="fantasy-readout__val">{props.points}</span>
        <span class="fantasy-readout__unit">PTS</span>
      </span>
    </span>
  )
}
