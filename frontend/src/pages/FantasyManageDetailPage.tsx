import { ConsoleCard } from '../components/ConsoleCard'
import { TeamScoreMeta } from '../components/FantasyScoreReadout'
import { Player } from '../components/Player'
import { RosterPlayerChip } from '../components/RosterPlayerChip'
import { TeamEditor } from '../components/TeamEditor'
import { For, Match, Show, Switch, createEffect, createMemo, createResource, createSignal, type JSX } from 'solid-js'
import { A } from '@solidjs/router'
import { authFetch } from '../lib/auth'
import { displayValue } from '../types/tournament'
import type { AuthUser } from '../types/user'
import {
  POINT_STAGES,
  sortByElo,
  type FantasyLeague,
  type FantasyPlayerRow,
  type FantasyTeamRow,
  type PointStage,
} from '../types/fantasy'

type Props = { leagueId: number }

type PlayerDraft = {
  cost: number
  pointsRo24: number | null
  pointsRo16: number | null
  pointsRo8: number | null
  pointsRo4: number | null
  pointsRo2: number | null
  defeated: boolean
  isWinner: boolean
}

async function fetchLeague(id: number): Promise<FantasyLeague> {
  const res = await authFetch(`/api/fantasy-leagues/${id}`)
  if (!res.ok) throw new Error(`league failed (${res.status})`)
  const data = (await res.json()) as { league: FantasyLeague }
  return data.league
}

async function fetchPlayers(id: number): Promise<FantasyPlayerRow[]> {
  const res = await authFetch(`/api/fantasy-leagues/${id}/players?sort=elo`)
  if (!res.ok) throw new Error(`players failed (${res.status})`)
  const data = (await res.json()) as { players: FantasyPlayerRow[] }
  return data.players ?? []
}

async function fetchTeams(id: number): Promise<FantasyTeamRow[]> {
  const res = await authFetch(`/api/fantasy-leagues/${id}/teams`)
  if (!res.ok) throw new Error(`teams failed (${res.status})`)
  const data = (await res.json()) as { teams: FantasyTeamRow[] }
  return data.teams ?? []
}

async function fetchUsers(): Promise<AuthUser[]> {
  const res = await authFetch('/api/users')
  if (!res.ok) throw new Error(`users failed (${res.status})`)
  const data = (await res.json()) as { users: AuthUser[] }
  return data.users ?? []
}

function toDraft(p: FantasyPlayerRow): PlayerDraft {
  return {
    cost: p.cost,
    pointsRo24: p.pointsRo24,
    pointsRo16: p.pointsRo16,
    pointsRo8: p.pointsRo8,
    pointsRo4: p.pointsRo4,
    pointsRo2: p.pointsRo2,
    defeated: p.defeated,
    isWinner: p.isWinner,
  }
}

function stageField(
  stage: PointStage,
): 'pointsRo24' | 'pointsRo16' | 'pointsRo8' | 'pointsRo4' | 'pointsRo2' {
  switch (stage) {
    case 'Ro24':
      return 'pointsRo24'
    case 'Ro16':
      return 'pointsRo16'
    case 'Ro8':
      return 'pointsRo8'
    case 'Ro4':
      return 'pointsRo4'
    case 'Ro2':
      return 'pointsRo2'
  }
}

async function readErr(res: Response): Promise<string> {
  try {
    const data = (await res.json()) as { error?: string }
    if (data.error) return data.error
  } catch {
    /* ignore */
  }
  return `request failed (${res.status})`
}

/** Admin detail for one fantasy league. */
export function FantasyManageDetailPage(props: Props): JSX.Element {
  const leagueId = () => props.leagueId
  const [league, { refetch: refetchLeague }] = createResource(leagueId, fetchLeague)
  const [players, { refetch: refetchPlayers }] = createResource(leagueId, fetchPlayers)
  const [teams, { refetch: refetchTeams }] = createResource(leagueId, fetchTeams)
  const [users] = createResource(fetchUsers)

  const [maxPlayers, setMaxPlayers] = createSignal(6)
  const [maxCost, setMaxCost] = createSignal(28)
  const [drafts, setDrafts] = createSignal<Record<number, PlayerDraft>>({})
  const [msg, setMsg] = createSignal<string | null>(null)
  const [err, setErr] = createSignal<string | null>(null)
  const [busy, setBusy] = createSignal(false)

  const [addUserId, setAddUserId] = createSignal<number | null>(null)
  const [addIds, setAddIds] = createSignal<number[]>([])
  const [editingTeamId, setEditingTeamId] = createSignal<number | null>(null)
  const [editIds, setEditIds] = createSignal<number[]>([])

  const orderedPlayers = createMemo(() => sortByElo(players() ?? []))

  const availableUsers = createMemo(() => {
    const taken = new Set((teams() ?? []).map((t) => t.userId))
    return (users() ?? []).filter((u) => !taken.has(u.id))
  })

  createEffect(() => {
    const l = league()
    if (l) {
      setMaxPlayers(l.maxPlayers)
      setMaxCost(l.maxCost)
    }
  })

  createEffect(() => {
    const rows = players()
    if (!rows) return
    const next: Record<number, PlayerDraft> = {}
    for (const p of rows) next[p.id] = toDraft(p)
    setDrafts(next)
  })

  function patchDraft(id: number, patch: Partial<PlayerDraft>) {
    setDrafts((prev) => ({ ...prev, [id]: { ...prev[id], ...patch } }))
  }

  async function saveCaps() {
    setBusy(true)
    setErr(null)
    setMsg(null)
    try {
      const res = await authFetch(`/api/fantasy-leagues/${leagueId()}`, {
        method: 'PATCH',
        body: JSON.stringify({ maxPlayers: maxPlayers(), maxCost: maxCost() }),
      })
      if (!res.ok) throw new Error(await readErr(res))
      setMsg('Caps saved')
      await refetchLeague()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Save failed')
    } finally {
      setBusy(false)
    }
  }

  async function startLeague() {
    setBusy(true)
    setErr(null)
    try {
      const res = await authFetch(`/api/fantasy-leagues/${leagueId()}/start`, { method: 'POST' })
      if (!res.ok) throw new Error(await readErr(res))
      setMsg('League started')
      await refetchLeague()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Start failed')
    } finally {
      setBusy(false)
    }
  }

  async function finishLeague() {
    setBusy(true)
    setErr(null)
    try {
      const res = await authFetch(`/api/fantasy-leagues/${leagueId()}/finish`, { method: 'POST' })
      if (!res.ok) throw new Error(await readErr(res))
      setMsg('League finished')
      await refetchLeague()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Finish failed')
    } finally {
      setBusy(false)
    }
  }

  async function savePlayers() {
    setBusy(true)
    setErr(null)
    setMsg(null)
    try {
      const rows = players() ?? []
      const d = drafts()
      for (const p of rows) {
        const draft = d[p.id]
        if (!draft) continue
        const res = await authFetch(`/api/fantasy-leagues/${leagueId()}/players/${p.id}`, {
          method: 'PATCH',
          body: JSON.stringify({
            cost: draft.cost,
            pointsRo24: draft.pointsRo24,
            pointsRo16: draft.pointsRo16,
            pointsRo8: draft.pointsRo8,
            pointsRo4: draft.pointsRo4,
            pointsRo2: draft.pointsRo2,
            defeated: draft.defeated,
            isWinner: draft.isWinner,
          }),
        })
        if (!res.ok) throw new Error(await readErr(res))
      }
      setMsg('Players saved')
      await refetchPlayers()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Save players failed')
    } finally {
      setBusy(false)
    }
  }

  async function createTeam() {
    const uid = addUserId()
    if (uid == null) return
    setBusy(true)
    setErr(null)
    try {
      const res = await authFetch(`/api/fantasy-leagues/${leagueId()}/teams`, {
        method: 'POST',
        body: JSON.stringify({ userId: uid, fantasyPlayerIds: addIds() }),
      })
      if (!res.ok) throw new Error(await readErr(res))
      setAddUserId(null)
      setAddIds([])
      setMsg('Team created')
      await refetchTeams()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Create team failed')
    } finally {
      setBusy(false)
    }
  }

  async function saveEditTeam() {
    const tid = editingTeamId()
    if (tid == null) return
    setBusy(true)
    setErr(null)
    try {
      const res = await authFetch(`/api/fantasy-leagues/${leagueId()}/teams/${tid}`, {
        method: 'PUT',
        body: JSON.stringify({ fantasyPlayerIds: editIds() }),
      })
      if (!res.ok) throw new Error(await readErr(res))
      setEditingTeamId(null)
      setEditIds([])
      setMsg('Team updated')
      await refetchTeams()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Update team failed')
    } finally {
      setBusy(false)
    }
  }

  async function deleteTeam(teamId: number) {
    if (!window.confirm('Delete this fantasy team?')) return
    setBusy(true)
    setErr(null)
    try {
      const res = await authFetch(`/api/fantasy-leagues/${leagueId()}/teams/${teamId}`, {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error(await readErr(res))
      if (editingTeamId() === teamId) {
        setEditingTeamId(null)
        setEditIds([])
      }
      setMsg('Team deleted')
      await refetchTeams()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Delete failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <ConsoleCard class="console--wide">
      <p class="status status--idle">
        <A href="/fantasy-manage">← Leagues</A>
      </p>
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Admin · League Detail</p>
        <h1 class="brand__title">
          <Switch>
            <Match when={league.loading}>Loading…</Match>
            <Match when={league()}>
              {(l) => <>{displayValue(l().tournamentName) || l().tournamentLink}</>}
            </Match>
          </Switch>
        </h1>
      </header>
      <hr class="rule" />

      <Show when={err()}>
        <p class="status status--error">{err()}</p>
      </Show>
      <Show when={msg()}>
        <p class="status status--ok">{msg()}</p>
      </Show>

      <Show when={league()}>
        {(l) => (
          <>
            <div class="fantasy-league-banner">
              <Show when={l().finished}>
                <span class="chip chip--alert">Finished</span>
              </Show>
              <Show when={l().started && !l().finished}>
                <span class="chip chip--live">Started · live</span>
              </Show>
              <Show when={!l().started && !l().finished}>
                <span class="chip fantasy-status-chip--open">Open · not started</span>
              </Show>
              <span class="fantasy-league-banner__caps">
                {l().maxPlayers} players · {l().maxCost} cost
              </span>
            </div>

            <div class="actions fantasy-detail__actions">
              <Show when={!l().started}>
                <button type="button" class="btn btn--primary" disabled={busy()} onClick={() => void startLeague()}>
                  Start league
                </button>
              </Show>
              <Show when={l().started && !l().finished}>
                <button type="button" class="btn btn--ghost" disabled={busy()} onClick={() => void finishLeague()}>
                  Finish league
                </button>
              </Show>
            </div>

            <Show when={!l().started}>
              <div class="fantasy-create__row">
                <label class="field">
                  <span class="field__label">Max players</span>
                  <span class="field__shell">
                    <input
                      class="field__input"
                      type="number"
                      min={1}
                      value={maxPlayers()}
                      onInput={(e) => setMaxPlayers(Number(e.currentTarget.value) || 1)}
                    />
                  </span>
                </label>
                <label class="field">
                  <span class="field__label">Max cost</span>
                  <span class="field__shell">
                    <input
                      class="field__input"
                      type="number"
                      min={1}
                      value={maxCost()}
                      onInput={(e) => setMaxCost(Number(e.currentTarget.value) || 1)}
                    />
                  </span>
                </label>
                <button type="button" class="btn btn--primary" disabled={busy()} onClick={() => void saveCaps()}>
                  Save caps
                </button>
              </div>
            </Show>
          </>
        )}
      </Show>

      <div class="fantasy-section-head">
        <h2 class="fantasy-section-title">Players</h2>
        <button type="button" class="btn btn--primary" disabled={busy()} onClick={() => void savePlayers()}>
          {busy() ? 'Saving…' : 'Save players'}
        </button>
      </div>

      <Switch>
        <Match when={players.loading}>
          <p class="status status--idle">Loading players…</p>
        </Match>
        <Match when={orderedPlayers().length > 0}>
          <div class="fantasy-admin-players">
            <For each={orderedPlayers()}>
              {(p) => {
                const d = () => drafts()[p.id] ?? toDraft(p)
                return (
                  <div
                    classList={{
                      'fantasy-admin-player': true,
                      'fantasy-admin-player--defeated': d().defeated,
                      'fantasy-admin-player--winner': d().isWinner,
                    }}
                  >
                    <div class="fantasy-admin-player__head">
                      <Player name={p.name} link={p.link} race={p.race} />
                      <div class="fantasy-admin-player__flags">
                        <Show when={d().isWinner}>
                          <span class="chip chip--live fantasy-chip">Champion</span>
                        </Show>
                        <Show when={d().defeated}>
                          <span class="chip chip--alert fantasy-chip">Defeated</span>
                        </Show>
                        <span class="fantasy-admin-player__meta">
                          elo {(p.elo ?? 0).toFixed(0)}
                        </span>
                      </div>
                    </div>
                    <div class="fantasy-create__row">
                      <label class="field">
                        <span class="field__label">Cost</span>
                        <span class="field__shell">
                          <input
                            class="field__input"
                            type="number"
                            min={0}
                            value={d().cost}
                            onInput={(e) =>
                              patchDraft(p.id, { cost: Number(e.currentTarget.value) || 0 })
                            }
                          />
                        </span>
                      </label>
                      <For each={[...POINT_STAGES]}>
                        {(stage) => {
                          const key = stageField(stage)
                          return (
                            <label class="field">
                              <span class="field__label">{stage}</span>
                              <span class="field__shell">
                                <input
                                  class="field__input"
                                  type="number"
                                  value={d()[key] ?? ''}
                                  placeholder="—"
                                  onInput={(e) => {
                                    const raw = e.currentTarget.value.trim()
                                    patchDraft(p.id, {
                                      [key]: raw === '' ? null : Number(raw),
                                    } as Partial<PlayerDraft>)
                                  }}
                                />
                              </span>
                            </label>
                          )
                        }}
                      </For>
                    </div>
                    <div class="actions">
                      <button
                        type="button"
                        class="btn btn--ghost"
                        onClick={() => patchDraft(p.id, { defeated: !d().defeated })}
                      >
                        {d().defeated ? 'Clear defeated' : 'Mark defeated'}
                      </button>
                      <button
                        type="button"
                        class="btn btn--ghost"
                        onClick={() => patchDraft(p.id, { isWinner: !d().isWinner })}
                      >
                        {d().isWinner ? 'Clear champion' : 'Mark champion'}
                      </button>
                    </div>
                  </div>
                )
              }}
            </For>
          </div>
        </Match>
      </Switch>

      <h2 class="fantasy-section-title">Teams</h2>
      <Show when={league()}>
        {(l) => (
          <div class="fantasy-admin-add-team">
            <label class="field">
              <span class="field__label">Add team for user</span>
              <span class="field__shell">
                <select
                  class="field__input"
                  value={addUserId() ?? ''}
                  onChange={(e) => {
                    const v = Number(e.currentTarget.value)
                    setAddUserId(Number.isFinite(v) && v > 0 ? v : null)
                  }}
                >
                  <option value="">Select user…</option>
                  <For each={availableUsers()}>{(u) => <option value={u.id}>{u.alias}</option>}</For>
                </select>
              </span>
            </label>
            <Show when={addUserId() != null}>
              <TeamEditor
                players={orderedPlayers()}
                selectedIds={addIds()}
                maxPlayers={l().maxPlayers}
                maxCost={l().maxCost}
                onChange={setAddIds}
              />
              <button type="button" class="btn btn--primary" disabled={busy()} onClick={() => void createTeam()}>
                Add team
              </button>
            </Show>
          </div>
        )}
      </Show>

      <Switch>
        <Match when={teams.loading}>
          <p class="status status--idle">Loading teams…</p>
        </Match>
        <Match when={(teams() ?? []).length === 0}>
          <p class="status status--idle">No teams yet</p>
        </Match>
        <Match when={teams()}>
          {(list) => (
            <ul class="fantasy-teams">
              <For each={list()}>
                {(t) => (
                  <li class="fantasy-team">
                    <header class="fantasy-team__head">
                      <div class="fantasy-team__identity">
                        <span class="fantasy-team__rank" data-rank={t.rank}>
                          #{t.rank}
                        </span>
                        <h3 class="fantasy-team__title">{t.userAlias}</h3>
                      </div>
                      <TeamScoreMeta
                        players={t.members.length}
                        points={t.points}
                        cost={t.cost}
                      />
                    </header>
                    <Show
                      when={editingTeamId() === t.id}
                      fallback={
                        <>
                          <div class="fantasy-team__bay">
                            <ul class="fantasy-team__list">
                              <For each={t.members}>
                                {(m) => (
                                  <li class="fantasy-team__slot">
                                    <RosterPlayerChip
                                      name={m.name}
                                      link={m.link}
                                      race={m.race}
                                      cost={m.cost}
                                      points={m.pointsEarned}
                                      defeated={m.defeated}
                                      isWinner={m.isWinner}
                                    />
                                  </li>
                                )}
                              </For>
                            </ul>
                          </div>
                          <div class="actions">
                            <button
                              type="button"
                              class="btn btn--ghost"
                              disabled={busy()}
                              onClick={() => {
                                setEditingTeamId(t.id)
                                setEditIds(t.members.map((m) => m.fantasyPlayerId))
                              }}
                            >
                              Edit team
                            </button>
                            <button
                              type="button"
                              class="btn btn--ghost"
                              disabled={busy()}
                              onClick={() => void deleteTeam(t.id)}
                            >
                              Delete
                            </button>
                          </div>
                        </>
                      }
                    >
                      <Show when={league()}>
                        {(l) => (
                          <>
                            <TeamEditor
                              players={orderedPlayers()}
                              selectedIds={editIds()}
                              maxPlayers={l().maxPlayers}
                              maxCost={l().maxCost}
                              disabled={busy()}
                              onChange={setEditIds}
                            />
                            <div class="actions">
                              <button
                                type="button"
                                class="btn btn--primary"
                                disabled={busy()}
                                onClick={() => void saveEditTeam()}
                              >
                                Save team
                              </button>
                              <button
                                type="button"
                                class="btn btn--ghost"
                                disabled={busy()}
                                onClick={() => {
                                  setEditingTeamId(null)
                                  setEditIds([])
                                }}
                              >
                                Cancel
                              </button>
                            </div>
                          </>
                        )}
                      </Show>
                    </Show>
                  </li>
                )}
              </For>
            </ul>
          )}
        </Match>
      </Switch>
    </ConsoleCard>
  )
}
