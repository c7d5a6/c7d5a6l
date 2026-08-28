import { ConsoleCard } from '../components/ConsoleCard'
import { ChannelHead } from '../components/ChannelChrome'
import { Player } from '../components/Player'
import { A } from '@solidjs/router'
import { For, Match, Show, Switch, createResource, createSignal } from 'solid-js'
import { authFetch, isAdmin } from '../lib/auth'
import { invalidatePlayerInfo } from '../lib/playerHoverCache'
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

function formatSeasonDate(iso: string): string {
  const d = iso.slice(0, 10)
  return /^\d{4}-\d{2}-\d{2}$/.test(d) ? d : iso
}

function formatRankDelta(delta: number | null | undefined): string {
  if (delta == null) return '—'
  if (delta === 0) return '0'
  return delta > 0 ? `+${delta}` : String(delta)
}

/** Roster channel — player_race rows ranked by elo. */
export function PlayersPage() {
  const [roster, { refetch }] = createResource(fetchPlayers)
  const [editingId, setEditingId] = createSignal<number | null>(null)
  const [draftElo, setDraftElo] = createSignal('')
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

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
          {(rows) => (
            <>
              <Show when={rows().length === 0}>
                <p class="status status--idle">No race entries in database</p>
              </Show>
              <Show when={rows().length > 0}>
                <Show when={roster()?.season}>
                  {(season) => (
                    <div class="season-strip">
                      <div class="season-strip__main">
                        <span class="season-strip__name">{season().name}</span>
                        <span class="season-strip__meta">
                          Opened {formatSeasonDate(season().startedAt)}
                        </span>
                      </div>
                      <Show when={season().readyToClose && isAdmin()}>
                        <A
                          href="/season-close"
                          class="chip chip--alert season-strip__chip season-strip__link"
                        >
                          Close season →
                        </A>
                      </Show>
                    </div>
                  )}
                </Show>

                <p class="status status--ok">
                  {rows().length} race entr{rows().length === 1 ? 'y' : 'ies'} · ranked by season rating
                </p>
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
                      Elo
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
                  <For each={rows()}>
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
                        </span>
                        <span class="roster__cell roster__elo" role="cell">
                          <Show
                            when={isAdmin() && editingId() === row.playerRaceId}
                            fallback={
                              <>
                                {formatElo(row.elo)}
                                <Show when={row.seasonStartElo != null}>
                                  <span class="roster__elo-start">
                                    {formatElo(row.seasonStartElo!)}
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
                              aria-label={`Elo for ${row.name ?? row.link}`}
                              onInput={(e) => setDraftElo(e.currentTarget.value)}
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') void saveEdit(row)
                                if (e.key === 'Escape') cancelEdit()
                              }}
                            />
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
                                <button
                                  type="button"
                                  class="btn btn--ghost btn--compact"
                                  disabled={
                                    busy() ||
                                    (editingId() != null && editingId() !== row.playerRaceId)
                                  }
                                  onClick={() => startEdit(row)}
                                >
                                  Edit
                                </button>
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
            </>
          )}
        </Match>
      </Switch>
      </div>
    </ConsoleCard>
  )
}
