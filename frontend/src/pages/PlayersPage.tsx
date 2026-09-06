import { ConsoleCard } from '../components/ConsoleCard'
import { ChannelHead } from '../components/ChannelChrome'
import { Player } from '../components/Player'
import { PlayerMergeModal } from '../components/PlayerMergeModal'
import { A } from '@solidjs/router'
import {
  For,
  Match,
  Show,
  Switch,
  createEffect,
  createMemo,
  createResource,
  createSignal,
} from 'solid-js'
import { authFetch, authUser, isAdmin } from '../lib/auth'
import { fetchActiveFantasyLeague, fetchFantasyPlayers, fetchMyFantasyTeam } from '../lib/api/fantasy'
import { invalidatePlayerInfo } from '../lib/playerHoverCache'
import { playerRaceKey } from '../lib/playerLink'
import type { FantasyPlayerRow, FantasyTeamRow } from '../types/fantasy'
import {
  playerPortraitSrc,
  type PlayerRaceEntry,
  type SeasonSummary,
} from '../types/tournament'

type ListPlayersResponse = {
  players: PlayerRaceEntry[]
  season?: SeasonSummary | null
}

type RosterPayload = {
  players: PlayerRaceEntry[]
  season: SeasonSummary | null
}

async function fetchPlayers(): Promise<RosterPayload> {
  const res = await authFetch('/api/players')
  if (!res.ok) {
    throw new Error(`roster uplink failed (${res.status})`)
  }
  const data = (await res.json()) as ListPlayersResponse
  return {
    players: data.players ?? [],
    season: data.season ?? null,
  }
}

function formatElo(elo: number): string {
  return elo.toFixed(0)
}

function displayElo(row: PlayerRaceEntry): number {
  return row.projectedElo ?? row.elo
}

function formatSeasonDate(iso: string): string {
  const d = iso.slice(0, 10)
  return /^\d{4}-\d{2}-\d{2}$/.test(d) ? d : iso
}

function formatRankDelta(delta: number | null | undefined): string {
  if (delta == null) return '—'
  if (delta === 0) return '0'
  return delta > 0 ? `+${delta}` : String(delta)
}

function fantasyRosterKeys(players: FantasyPlayerRow[]): Set<string> {
  const keys = new Set<string>()
  for (const p of players) {
    if (p.link && p.race) keys.add(playerRaceKey(p.link, p.race))
  }
  return keys
}

function teamRosterKeys(team: FantasyTeamRow | null | undefined): Set<string> {
  const keys = new Set<string>()
  if (!team) return keys
  for (const m of team.members) {
    if (m.link && m.race) keys.add(playerRaceKey(m.link, m.race))
  }
  return keys
}

/** Roster channel — player_race rows ranked by elo. */
export function PlayersPage() {
  const [roster, { refetch }] = createResource(fetchPlayers)
  const [activeLeague] = createResource(fetchActiveFantasyLeague)
  const [fantasyPlayers] = createResource(
    () => activeLeague()?.id ?? null,
    (id) => (id == null ? Promise.resolve([] as FantasyPlayerRow[]) : fetchFantasyPlayers(id)),
  )
  const myTeamLeagueId = createMemo(() => {
    if (!authUser() || !activeLeague()?.id) return null
    return activeLeague()!.id
  })
  const [myTeam] = createResource(myTeamLeagueId, (id) => fetchMyFantasyTeam(id))
  const [fantasyOnly, setFantasyOnly] = createSignal(false)
  const [editingId, setEditingId] = createSignal<number | null>(null)
  const [draftElo, setDraftElo] = createSignal('')
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)
  const [mergeMain, setMergeMain] = createSignal<PlayerRaceEntry | null>(null)

  const allRows = createMemo(() => roster()?.players ?? [])
  const fantasyKeys = createMemo(() => fantasyRosterKeys(fantasyPlayers() ?? []))
  const myTeamKeys = createMemo(() => teamRosterKeys(myTeam()))
  const visibleRows = createMemo(() => {
    const rows = allRows()
    if (!fantasyOnly()) return rows
    const keys = fantasyKeys()
    return rows.filter((row) => keys.has(playerRaceKey(row.link, row.race)))
  })

  createEffect(() => {
    if (!activeLeague() && fantasyOnly()) setFantasyOnly(false)
  })

  function startEdit(row: PlayerRaceEntry) {
    setEditingId(row.playerRaceId)
    setDraftElo(formatElo(row.elo))
    setError(null)
  }

  function cancelEdit() {
    setEditingId(null)
    setDraftElo('')
    setError(null)
  }

  async function saveEdit(row: PlayerRaceEntry) {
    const n = Number(draftElo())
    if (!Number.isFinite(n) || n < 0 || n > 9999) {
      setError('Elo must be between 0 and 9999')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const res = await authFetch(`/api/players/races/${row.playerRaceId}`, {
        method: 'PATCH',
        body: JSON.stringify({ elo: Math.round(n) }),
      })
      if (!res.ok) {
        let msg = `save failed (${res.status})`
        try {
          const data = (await res.json()) as { error?: string }
          if (data.error) msg = data.error
        } catch {
          /* ignore */
        }
        throw new Error(msg)
      }
      invalidatePlayerInfo(row.link)
      cancelEdit()
      await refetch()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <ConsoleCard>
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Roster · Command Protocol</p>
        <h1 class="brand__title">
          Players <span>Channel</span>
        </h1>
      </header>
      <hr class="rule" />

      <div class="channel-stack">
        <ChannelHead tag="Roster" title="Players" />
      <Switch>
        <Match when={roster.loading}>
          <p class="status status--idle">Locking roster uplink…</p>
        </Match>
        <Match when={roster.error}>
          <p class="status status--error">
            {(roster.error as Error)?.message ?? 'Roster uplink failed'}
          </p>
        </Match>
        <Match when={roster()?.players}>
          <>
              <Show when={allRows().length === 0}>
                <p class="status status--idle">No race entries in database</p>
              </Show>
              <Show when={allRows().length > 0}>
                <Show when={roster()?.season}>
                  {(season) => (
                    <div class="season-strip">
                      <div class="season-strip__main">
                        <span class="season-strip__name">{season().name}</span>
                        <span class="season-strip__meta">
                          Opened {formatSeasonDate(season().startedAt)}
                        </span>
                      </div>
                      <div class="season-strip__actions">
                        <Show when={activeLeague()}>
                          <button
                            type="button"
                            class="chip chip--compact season-strip__chip"
                            classList={{ 'chip--on': fantasyOnly() }}
                            aria-pressed={fantasyOnly()}
                            disabled={fantasyPlayers.loading}
                            onClick={() => setFantasyOnly((on) => !on)}
                          >
                            {fantasyOnly() ? 'Show all players' : 'Show fantasy roster'}
                          </button>
                        </Show>
                        <Show when={isAdmin()}>
                          <A href="/season-close" class="chip chip--compact season-strip__chip season-strip__link">
                            Close season
                          </A>
                        </Show>
                      </div>
                    </div>
                  )}
                </Show>

                <Show when={!roster()?.season && activeLeague()}>
                  {(league) => (
                    <div class="season-strip">
                      <div class="season-strip__main">
                        <span class="season-strip__name">Fantasy league</span>
                        <span class="season-strip__meta">
                          {league().tournamentName ?? league().tournamentLink}
                        </span>
                      </div>
                      <button
                        type="button"
                        class="chip chip--compact season-strip__chip"
                        classList={{ 'chip--on': fantasyOnly() }}
                        aria-pressed={fantasyOnly()}
                        disabled={fantasyPlayers.loading}
                        onClick={() => setFantasyOnly((on) => !on)}
                      >
                        {fantasyOnly() ? 'Show all players' : 'Show fantasy roster'}
                      </button>
                    </div>
                  )}
                </Show>

                <Show when={fantasyOnly() && visibleRows().length === 0}>
                  <p class="status status--idle">No players in the current fantasy league roster</p>
                </Show>

                <p class="status status--ok">
                  {fantasyOnly()
                    ? `${visibleRows().length} of ${allRows().length} race entr${allRows().length === 1 ? 'y' : 'ies'}`
                    : `${allRows().length} race entr${allRows().length === 1 ? 'y' : 'ies'}`}{' '}
                  · ranked by season rating
                </p>
                <Show when={visibleRows().length > 0}>
                <Show when={error()}>
                  <p class="status status--error">{error()}</p>
                </Show>
                <div
                  classList={{
                    roster: true,
                    'roster--players': true,
                    'roster--players-admin': isAdmin(),
                  }}
                  role="table"
                  aria-label="Players by elo"
                >
                  <div class="roster__head" role="row">
                    <span class="roster__cell roster__rank" role="columnheader">
                      #
                    </span>
                    <span class="roster__cell roster__player" role="columnheader">
                      Player
                    </span>
                    <span class="roster__cell roster__elo" role="columnheader">
                      Rating
                    </span>
                    <span class="roster__cell roster__rank-delta" role="columnheader">
                      Δ Rank
                    </span>
                    <Show when={isAdmin()}>
                      <span class="roster__cell roster__actions" role="columnheader">
                        <span class="points-board__sr-only">Actions</span>
                      </span>
                    </Show>
                  </div>
                  <For each={visibleRows()}>
                    {(row, i) => (
                      <div class="roster__row" role="row">
                        <span class="roster__cell roster__rank" role="cell">
                          {i() + 1}
                        </span>
                        <span class="roster__cell roster__player" role="cell">
                          <Show when={playerPortraitSrc(row)}>
                            {(src) => <img class="roster__portrait" src={src()} alt="" />}
                          </Show>
                          <Player
                            name={row.name}
                            link={row.link}
                            race={row.race}
                            hasPortrait={row.hasPortrait}
                          />
                          <Show when={myTeamKeys().has(playerRaceKey(row.link, row.race))}>
                            <span class="chip chip--compact chip--fantasy roster__team-tag">Team</span>
                          </Show>
                        </span>
                        <span class="roster__cell roster__elo" role="cell">
                          <Show
                            when={isAdmin() && editingId() === row.playerRaceId}
                            fallback={
                              <>
                                {formatElo(displayElo(row))}
                                <Show when={row.lastSeasonEndElo != null}>
                                  <span class="roster__elo-start">
                                    last {formatElo(row.lastSeasonEndElo!)}
                                  </span>
                                </Show>
                              </>
                            }
                          >
                            <input
                              class="field__input fantasy-cost-input roster__elo-input"
                              type="number"
                              min={0}
                              max={9999}
                              step={1}
                              value={draftElo()}
                              disabled={busy()}
                              aria-label={`Season start rating for ${row.name ?? row.link}`}
                              onInput={(e) => setDraftElo(e.currentTarget.value)}
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') void saveEdit(row)
                                if (e.key === 'Escape') cancelEdit()
                              }}
                            />
                            <span class="roster__elo-start">season start</span>
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
                        <Show when={isAdmin()}>
                          <span class="roster__cell roster__actions" role="cell">
                            <Show
                              when={editingId() === row.playerRaceId}
                              fallback={
                                <div class="roster__elo-actions">
                                  <button
                                    type="button"
                                    class="btn btn--ghost btn--compact"
                                    disabled={
                                      busy() ||
                                      mergeMain() != null ||
                                      (editingId() != null && editingId() !== row.playerRaceId)
                                    }
                                    onClick={() => startEdit(row)}
                                  >
                                    Edit
                                  </button>
                                  <button
                                    type="button"
                                    class="btn btn--ghost btn--compact"
                                    disabled={
                                      busy() ||
                                      mergeMain() != null ||
                                      (editingId() != null && editingId() !== row.playerRaceId)
                                    }
                                    onClick={() => {
                                      setError(null)
                                      setMergeMain(row)
                                    }}
                                  >
                                    Merge
                                  </button>
                                </div>
                              }
                            >
                              <div class="roster__elo-actions">
                                <button
                                  type="button"
                                  class="btn btn--primary btn--compact"
                                  disabled={busy()}
                                  onClick={() => void saveEdit(row)}
                                >
                                  {busy() ? 'Saving…' : 'Save'}
                                </button>
                                <button
                                  type="button"
                                  class="btn btn--ghost btn--compact"
                                  disabled={busy()}
                                  onClick={cancelEdit}
                                >
                                  Cancel
                                </button>
                              </div>
                            </Show>
                          </span>
                        </Show>
                      </div>
                    )}
                  </For>
                </div>
                </Show>
              </Show>
          </>
        </Match>
      </Switch>
      </div>

      <Show when={mergeMain()}>
        {(main) => (
          <PlayerMergeModal
            main={main()}
            onClose={() => setMergeMain(null)}
            onMerged={async () => {
              await refetch()
            }}
          />
        )}
      </Show>
    </ConsoleCard>
  )
}
