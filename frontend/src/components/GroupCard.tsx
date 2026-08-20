import { For, Show, type JSX } from 'solid-js'
import { Player } from './Player'
import type { Participant, Result } from '../types/tournament'
import { displayValue } from '../types/tournament'

export type MatchRowProps = {
  result: Result
  compact?: boolean
}

/** Two sides + score/vs + optional time. */
export function MatchRow(props: MatchRowProps): JSX.Element {
  const r = () => props.result
  const a = () => r().participantA
  const b = () => r().participantB
  const played = () =>
    r().played && r().scoreA != null && r().scoreB != null
  const loserA = () => played() && (r().scoreA as number) < (r().scoreB as number)
  const loserB = () => played() && (r().scoreB as number) < (r().scoreA as number)
  const winA = () => played() && (r().scoreA as number) > (r().scoreB as number)
  const winB = () => played() && (r().scoreB as number) > (r().scoreA as number)

  return (
    <div classList={{ 'match-row': true, 'match-row--compact': Boolean(props.compact) }}>
      <div class="match-row__side match-row__side--a">
        <Show when={a()} fallback={<span class="telemetry__val">—</span>}>
          {(p) => <SidePlayer p={p()} loser={Boolean(loserA())} />}
        </Show>
      </div>
      <span class="match-row__score">
        <Show when={r().played} fallback={<span class="match-row__vs">vs</span>}>
          <span classList={{ 'match-row__n': true, 'match-row__n--win': Boolean(winA()) }}>
            {displayValue(r().scoreA)}
          </span>
          <span class="match-row__sep">:</span>
          <span classList={{ 'match-row__n': true, 'match-row__n--win': Boolean(winB()) }}>
            {displayValue(r().scoreB)}
          </span>
        </Show>
      </span>
      <div class="match-row__side match-row__side--b">
        <Show when={b()} fallback={<span class="telemetry__val">—</span>}>
          {(p) => <SidePlayer p={p()} loser={Boolean(loserB())} />}
        </Show>
      </div>
      <Show when={r().dateTime}>
        <span class="match-row__time">{formatMatchTime(r().dateTime)}</span>
      </Show>
    </div>
  )
}

function SidePlayer(props: { p: Participant; loser: boolean }): JSX.Element {
  return (
    <Player
      name={props.p.name}
      link={props.p.link}
      race={props.p.race}
      excluded={props.p.excluded}
      loser={props.loser}
    />
  )
}

export function formatMatchTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export type GroupCardProps = {
  name: string
  phase?: string
  playoff?: boolean
  /** Tighter board layout (Results tab): metal plate, matches only. */
  dense?: boolean
  players?: Participant[]
  results?: Result[]
  /** Optional trailing chips (e.g. fantasy cost). */
  playerExtra?: (p: Participant, i: number) => JSX.Element
}

/** Bordered group shell with optional player chips and match list. */
export function GroupCard(props: GroupCardProps): JSX.Element {
  const hasResults = () => (props.results?.length ?? 0) > 0
  // Dense Results board: roster only as fallback when the group has no matches yet.
  const showPlayers = () =>
    (props.players?.length ?? 0) > 0 && (!props.dense || !hasResults())

  return (
    <article
      classList={{
        'group-card': true,
        'group-card--playoff': Boolean(props.playoff),
        'group-card--dense': Boolean(props.dense),
      }}
    >
      <header class="group-card__head">
        <span class="group-card__mark" aria-hidden="true" />
        <span class="group-card__name">{props.name}</span>
        <Show when={props.phase}>
          <span class="group-card__phase">{props.phase}</span>
        </Show>
      </header>
      <Show when={showPlayers()}>
        <ul class="telemetry__list telemetry__list--chips group-card__players">
          <For each={props.players}>
            {(p, i) => (
              <li class="group-card__player">
                <Player name={p.name} link={p.link} race={p.race} excluded={p.excluded} />
                {props.playerExtra?.(p, i())}
              </li>
            )}
          </For>
        </ul>
      </Show>
      <Show when={hasResults()}>
        <div class="group-card__matches">
          <For each={props.results}>{(m) => <MatchRow result={m} compact />}</For>
        </div>
      </Show>
      <Show when={!hasResults() && !showPlayers()}>
        <p class="group-card__empty">No matches</p>
      </Show>
    </article>
  )
}
