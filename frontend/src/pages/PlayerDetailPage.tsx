import { A } from '@solidjs/router'
import {
  For,
  Match,
  Show,
  Switch,
  createResource,
  type JSX,
} from 'solid-js'
import { ConsoleCard } from '../components/ConsoleCard'
import { ChannelHead } from '../components/ChannelChrome'
import { Player } from '../components/Player'
import { fetchPlayer } from '../lib/api/players'
import { parseRaceId, RACE_META } from '../lib/races'
import {
  displayValue,
  playerPortraitSrc,
  type PlayerDetail,
  type PlayerMatch,
  type PlayerRaceEntry,
  type WinrateBlock,
  type WinrateStat,
} from '../types/tournament'

function formatElo(elo: number): string {
  return elo.toFixed(0)
}

function displayElo(row: PlayerRaceEntry): number {
  return row.projectedElo ?? row.elo
}

function formatRankDelta(delta: number | null | undefined): string {
  if (delta == null) return '—'
  if (delta === 0) return '0'
  return delta > 0 ? `+${delta}` : String(delta)
}

function formatWinrate(stat: WinrateStat): string {
  if (stat.rate == null) return '—'
  return `${stat.rate.toFixed(1)}%`
}

function formatMaps(stat: WinrateStat): string {
  if (stat.wins + stat.losses === 0) return '0–0'
  return `${stat.wins}–${stat.losses}`
}

/** Red (<40%) → yellow (50%) → green (>65%), linear between stops. */
function winrateColor(rate: number | null): string | undefined {
  if (rate == null) return undefined
  const red = [213, 1, 3] as const
  const yellow = [201, 162, 39] as const
  const green = [112, 254, 58] as const
  const rgb = (c: readonly [number, number, number]) => `rgb(${c[0]}, ${c[1]}, ${c[2]})`
  const lerp = (a: readonly [number, number, number], b: readonly [number, number, number], t: number) => {
    const u = Math.min(1, Math.max(0, t))
    return rgb([
      Math.round(a[0] + (b[0] - a[0]) * u),
      Math.round(a[1] + (b[1] - a[1]) * u),
      Math.round(a[2] + (b[2] - a[2]) * u),
    ])
  }
  if (rate <= 40) return rgb(red)
  if (rate >= 65) return rgb(green)
  if (rate <= 50) return lerp(red, yellow, (rate - 40) / 10)
  return lerp(yellow, green, (rate - 50) / 15)
}

function formatMatchTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = iso.slice(0, 16).replace('T', ' ')
  return d
}

const WINRATE_SCOPES: { key: keyof PlayerDetail['winrates']; label: string }[] = [
  { key: 'season', label: 'Current season' },
  { key: 'fantasy', label: 'Current fantasy' },
  { key: 'overall', label: 'Overall' },
  { key: 'allFantasy', label: 'All fantasy leagues' },
]

const WINRATE_VS: { key: keyof WinrateBlock; label: string }[] = [
  { key: 'vsAll', label: 'All' },
  { key: 'vsTerran', label: 'vs T' },
  { key: 'vsZerg', label: 'vs Z' },
  { key: 'vsProtoss', label: 'vs P' },
]

/** Player dossier — identity, race ratings, winrates, season matches. */
export function PlayerDetailPage(props: { playerId: number }): JSX.Element {
  const [data] = createResource(
    () => props.playerId,
    (id) => fetchPlayer(id),
  )

  return (
    <ConsoleCard class="console--wide">
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Roster · Command Protocol</p>
        <h1 class="brand__title">
          Player <span>Dossier</span>
        </h1>
      </header>
      <hr class="rule" />
      <p class="status status--idle">
        <A href="/players">← Players roster</A>
      </p>

      <Switch>
        <Match when={data.loading}>
          <p class="status status--idle">Locking player dossier…</p>
        </Match>
        <Match when={data.error}>
          <p class="status status--error">
            {(data.error as Error)?.message ?? 'Player uplink failed'}
          </p>
        </Match>
        <Match when={data()}>
          {(player) => <PlayerDossier player={player()} />}
        </Match>
      </Switch>
    </ConsoleCard>
  )
}

function PlayerDossier(props: { player: PlayerDetail }): JSX.Element {
  const p = () => props.player
  const portrait = () => playerPortraitSrc(p())
  const raceMeta = () => {
    const id = parseRaceId(p().preferredRace)
    return id ? RACE_META[id] : null
  }

  return (
    <div class="channel-stack player-dossier">
      <ChannelHead tag="Dossier" title={displayValue(p().name)} />

      <section class="player-dossier__identity" aria-label="Player identity">
        <div class="player-dossier__portrait">
          <Show when={portrait()} fallback={<div class="player-dossier__portrait-empty" aria-hidden="true" />}>
            {(src) => <img src={src()} alt="" />}
          </Show>
        </div>
        <dl class="player-dossier__meta">
          <div>
            <dt>Name</dt>
            <dd class="player-dossier__name-row">
              <Show when={raceMeta()}>
                {(m) => <img class="player-dossier__race" src={m().icon} alt="" title={m().label} />}
              </Show>
              <span>{displayValue(p().name)}</span>
            </dd>
          </div>
          <div>
            <dt>Real name</dt>
            <dd>{displayValue(p().realName)}</dd>
          </div>
          <div>
            <dt>Race</dt>
            <dd>{displayValue(p().preferredRace)}</dd>
          </div>
          <div>
            <dt>Link</dt>
            <dd>
              <a class="player-dossier__ext" href={p().link} target="_blank" rel="noreferrer">
                {p().link}
              </a>
            </dd>
          </div>
          <div>
            <dt>IDs</dt>
            <dd class="player-dossier__ids">
              <Show when={p().ids.length > 0} fallback="—">
                <For each={p().ids}>{(id) => <span class="player-dossier__id">{id}</span>}</For>
              </Show>
            </dd>
          </div>
        </dl>
      </section>

      <ChannelHead tag="Rating" title="Race ratings" compact />
      <Show when={p().races.length > 0} fallback={<p class="status status--idle">No race entries</p>}>
        <div class="roster roster--players" role="table" aria-label="Race ratings">
          <div class="roster__head" role="row">
            <span class="roster__cell roster__rank" role="columnheader">
              #
            </span>
            <span class="roster__cell roster__player" role="columnheader">
              Race
            </span>
            <span class="roster__cell roster__elo" role="columnheader">
              Rating
            </span>
            <span class="roster__cell roster__rank-delta" role="columnheader">
              Δ Rank
            </span>
          </div>
          <For each={p().races}>
            {(row) => (
              <div class="roster__row" role="row">
                <span class="roster__cell roster__rank" role="cell">
                  {row.rank ?? '—'}
                </span>
                <span class="roster__cell roster__player" role="cell">
                  <Show
                    when={parseRaceId(row.race)}
                    fallback={<span>{displayValue(row.race)}</span>}
                  >
                    {(id) => {
                      const m = () => RACE_META[id()]
                      return (
                        <span class="player">
                          <img class="player__icon" src={m().icon} alt="" title={m().label} />
                          <span class="player__name">{m().label}</span>
                        </span>
                      )
                    }}
                  </Show>
                </span>
                <span class="roster__cell roster__elo" role="cell">
                  {formatElo(displayElo(row))}
                  <Show when={row.lastSeasonEndElo != null}>
                    <span class="roster__elo-start">last {formatElo(row.lastSeasonEndElo!)}</span>
                  </Show>
                </span>
                <span
                  classList={{
                    'roster__cell': true,
                    'roster__rank-delta': true,
                    'roster__rank-delta--up': (row.rankDelta ?? 0) > 0,
                    'roster__rank-delta--down': (row.rankDelta ?? 0) < 0,
                  }}
                  role="cell"
                >
                  {formatRankDelta(row.rankDelta)}
                </span>
              </div>
            )}
          </For>
        </div>
      </Show>

      <ChannelHead tag="Maps" title="Winrate" compact />
      <div class="winrate-board" role="table" aria-label="Winrates by scope">
        <div class="winrate-board__head" role="row">
          <span class="winrate-board__cell winrate-board__scope" role="columnheader">
            Scope
          </span>
          <For each={WINRATE_VS}>
            {(col) => (
              <span class="winrate-board__cell" role="columnheader">
                {col.label}
              </span>
            )}
          </For>
        </div>
        <For each={WINRATE_SCOPES}>
          {(scope) => {
            const block = () => p().winrates[scope.key]
            return (
              <div class="winrate-board__row" role="row">
                <span class="winrate-board__cell winrate-board__scope" role="cell">
                  {scope.label}
                </span>
                <For each={WINRATE_VS}>
                  {(col) => {
                    const stat = () => block()[col.key]
                    return (
                      <span class="winrate-board__cell" role="cell" title={formatMaps(stat())}>
                        <span
                          class="winrate-board__rate"
                          style={{ color: winrateColor(stat().rate) }}
                        >
                          {formatWinrate(stat())}
                        </span>
                        <span class="winrate-board__maps">{formatMaps(stat())}</span>
                      </span>
                    )
                  }}
                </For>
              </div>
            )
          }}
        </For>
      </div>

      <ChannelHead tag="Season" title="Matches this season" compact />
      <Show
        when={p().seasonMatches.length > 0}
        fallback={<p class="status status--idle">No played matches in the current season window</p>}
      >
        <div class="player-dossier__matches" role="list">
          <For each={p().seasonMatches}>{(m) => <SeasonMatchRow match={m} />}</For>
        </div>
      </Show>
    </div>
  )
}

function SeasonMatchRow(props: { match: PlayerMatch }): JSX.Element {
  const m = () => props.match
  const won = () => m().scoreMine > m().scoreOpp
  const lost = () => m().scoreMine < m().scoreOpp
  const myRace = () => {
    const id = parseRaceId(m().myRace)
    return id ? RACE_META[id] : null
  }

  return (
    <div class="match-row match-row--compact player-match" role="listitem">
      <div class="match-row__side">
        <Player
          name={m().opponentName}
          link={m().opponentLink}
          race={m().opponentRace}
          playerId={m().opponentPlayerId}
        />
        <span class="player-match__tour">
          {displayValue(m().tournamentName) || m().tournamentLink}
          <Show when={m().phase || m().round}>
            {' · '}
            {[m().phase, m().round].filter(Boolean).join(' / ')}
          </Show>
        </span>
      </div>
      <span class="match-row__score" title="Your maps : opponent maps">
        <Show when={myRace()}>
          {(meta) => <img class="player-match__my-race" src={meta().icon} alt="" title={meta().label} />}
        </Show>
        <span classList={{ 'match-row__n': true, 'match-row__n--win': won(), 'match-row__n--lose': lost() }}>
          {m().scoreMine}
        </span>
        <span class="match-row__sep">:</span>
        <span classList={{ 'match-row__n': true, 'match-row__n--win': lost(), 'match-row__n--lose': won() }}>
          {m().scoreOpp}
        </span>
      </span>
      <div class="match-row__side match-row__side--b">
        <Show when={m().dateTime}>
          <span class="match-row__time">{formatMatchTime(m().dateTime)}</span>
        </Show>
      </div>
    </div>
  )
}
