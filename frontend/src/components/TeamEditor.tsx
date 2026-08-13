import { For, Show, createMemo, type JSX } from 'solid-js'
import { Player } from './Player'
import { displayValue } from '../types/tournament'
import { sortByElo, type FantasyPlayerRow } from '../types/fantasy'

export type TeamEditorProps = {
  players: FantasyPlayerRow[]
  selectedIds: number[]
  maxPlayers: number
  maxCost: number
  disabled?: boolean
  onChange: (ids: number[]) => void
}

/** Shared multi-select roster picker with live cap feedback (players sorted by elo). */
export function TeamEditor(props: TeamEditorProps): JSX.Element {
  const ordered = createMemo(() => sortByElo(props.players))
  const selected = createMemo(() => new Set(props.selectedIds))
  const selectedPlayers = createMemo(() =>
    sortByElo(props.players.filter((p) => selected().has(p.id))),
  )
  const count = createMemo(() => selectedPlayers().length)
  const costSum = createMemo(() => selectedPlayers().reduce((s, p) => s + p.cost, 0))
  const overCount = createMemo(() => count() > props.maxPlayers)
  const overCost = createMemo(() => costSum() > props.maxCost)

  function toggle(id: number) {
    if (props.disabled) return
    const next = new Set(props.selectedIds)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    props.onChange([...next])
  }

  return (
    <div class="team-editor">
      <header class="team-editor__head">
        <p
          classList={{
            status: true,
            'status--ok': !overCount() && !overCost(),
            'status--error': overCount() || overCost(),
          }}
        >
          {count()} / {props.maxPlayers} players · cost {costSum()} / {props.maxCost}
        </p>
        <div class="team-editor__picks" aria-live="polite" aria-label="Selected roster">
          <Show
            when={count() > 0}
            fallback={<span class="team-editor__picks-idle">No picks locked</span>}
          >
            <For each={selectedPlayers()}>
              {(p) => (
                <button
                  type="button"
                  class="team-editor__pick"
                  disabled={props.disabled}
                  title={`Remove ${displayValue(p.name)}`}
                  onClick={() => toggle(p.id)}
                >
                  {displayValue(p.name)}
                  <span class="team-editor__pick-cost">{p.cost}</span>
                </button>
              )}
            </For>
          </Show>
        </div>
      </header>
      <ul class="team-editor__list">
        <For each={ordered()}>
          {(p) => {
            const on = () => selected().has(p.id)
            return (
              <li class="team-editor__item">
                <label classList={{ 'team-editor__label': true, 'team-editor__label--on': on() }}>
                  <span class="sc-check">
                    <input
                      type="checkbox"
                      class="sc-check__input"
                      checked={on()}
                      disabled={props.disabled}
                      onChange={() => toggle(p.id)}
                    />
                    <span class="sc-check__box" aria-hidden="true" />
                  </span>
                  <Player name={p.name} link={p.link} race={p.race} />
                  <span class="team-editor__elo">{(p.elo ?? 0).toFixed(0)}</span>
                  <span class="team-editor__cost">{p.cost}</span>
                  <Show when={p.defeated}>
                    <span class="chip chip--alert fantasy-chip">out</span>
                  </Show>
                  <Show when={p.isWinner}>
                    <span class="chip chip--live fantasy-chip">champion</span>
                  </Show>
                </label>
              </li>
            )
          }}
        </For>
      </ul>
    </div>
  )
}
