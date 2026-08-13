import { ConsoleCard } from '../components/ConsoleCard'
import { Player } from '../components/Player'
import { For, Match, Show, Switch, createMemo, createResource, createSignal, type JSX } from 'solid-js'
import { useNavigate } from '@solidjs/router'
import { authFetch } from '../lib/auth'
import { displayValue } from '../types/tournament'
import type { FantasyLeague, FantasyPreviewPlayer, TournamentSummary } from '../types/fantasy'

async function fetchLeagues(): Promise<FantasyLeague[]> {
  const res = await fetch('/api/fantasy-leagues')
  if (!res.ok) throw new Error(`leagues uplink failed (${res.status})`)
  const data = (await res.json()) as { leagues: FantasyLeague[] }
  return data.leagues ?? []
}

async function fetchUnused(): Promise<TournamentSummary[]> {
  const res = await authFetch('/api/tournaments/unused-for-fantasy')
  if (!res.ok) throw new Error(`unused tournaments failed (${res.status})`)
  const data = (await res.json()) as { tournaments: TournamentSummary[] }
  return data.tournaments ?? []
}

async function fetchPreview(
  tournamentId: number,
  costMin: number,
  costMax: number,
): Promise<FantasyPreviewPlayer[]> {
  const q = new URLSearchParams({
    tournamentId: String(tournamentId),
    costMin: String(costMin),
    costMax: String(costMax),
  })
  const res = await authFetch(`/api/fantasy-leagues/preview?${q}`)
  if (!res.ok) throw new Error(`preview failed (${res.status})`)
  const data = (await res.json()) as { players: FantasyPreviewPlayer[] }
  return data.players ?? []
}

function leagueStatus(l: FantasyLeague): { label: string; kind: 'open' | 'live' | 'done' } {
  if (l.finished) return { label: 'Finished', kind: 'done' }
  if (l.started) return { label: 'Started', kind: 'live' }
  return { label: 'Open', kind: 'open' }
}

/** Admin fantasy league list + create wizard. */
export function FantasyManagePage(): JSX.Element {
  const navigate = useNavigate()
  const [leagues, { refetch: refetchLeagues }] = createResource(fetchLeagues)
  const [unused, { refetch: refetchUnused }] = createResource(fetchUnused)

  const [tournamentId, setTournamentId] = createSignal<number | null>(null)
  const [costMin, setCostMin] = createSignal(0)
  const [costMax, setCostMax] = createSignal(10)
  const [maxPlayers, setMaxPlayers] = createSignal(6)
  const [maxCost, setMaxCost] = createSignal(28)
  /** Baseline costs from last preview; overrides layered on top. */
  const [baseCosts, setBaseCosts] = createSignal<Record<number, number>>({})
  const [overrides, setOverrides] = createSignal<Record<number, number>>({})
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  const previewKey = createMemo(() => {
    const tid = tournamentId()
    if (tid == null) return null
    return { tid, costMin: costMin(), costMax: costMax() }
  })

  const [preview] = createResource(previewKey, async (k) => {
    const rows = await fetchPreview(k.tid, k.costMin, k.costMax)
    const next: Record<number, number> = {}
    for (const r of rows) next[r.tournamentPlayerId] = r.cost
    setBaseCosts(next)
    setOverrides({})
    return rows
  })

  const displayRows = createMemo(() => {
    const rows = preview() ?? []
    const base = baseCosts()
    const ov = overrides()
    return rows.map((r) => ({
      ...r,
      cost: ov[r.tournamentPlayerId] ?? base[r.tournamentPlayerId] ?? r.cost,
    }))
  })

  async function onCreate(e: Event) {
    e.preventDefault()
    const tid = tournamentId()
    if (tid == null) return
    setBusy(true)
    setError(null)
    try {
      // Persist exactly what the admin sees (WYSIWYG costs).
      const costs = displayRows().map((r) => ({
        tournamentPlayerId: r.tournamentPlayerId,
        cost: r.cost,
      }))
      const res = await authFetch('/api/fantasy-leagues', {
        method: 'POST',
        body: JSON.stringify({
          tournamentId: tid,
          maxPlayers: maxPlayers(),
          maxCost: maxCost(),
          costMin: costMin(),
          costMax: costMax(),
          costs,
        }),
      })
      if (!res.ok) {
        let msg = `create failed (${res.status})`
        try {
          const data = (await res.json()) as { error?: string }
          if (data.error) msg = data.error
        } catch {
          /* ignore */
        }
        throw new Error(msg)
      }
      const data = (await res.json()) as { league: FantasyLeague }
      setTournamentId(null)
      setOverrides({})
      setBaseCosts({})
      await Promise.all([refetchLeagues(), refetchUnused()])
      navigate(`/fantasy-manage/${data.league.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <ConsoleCard class="console--wide">
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Admin · Fantasy Protocol</p>
        <h1 class="brand__title">
          League <span>Manage</span>
        </h1>
      </header>
      <hr class="rule" />

      <form class="fantasy-create" onSubmit={(e) => void onCreate(e)}>
        <label class="field">
          <span class="field__label">Unused tournament</span>
          <span class="field__shell">
            <select
              class="field__input"
              value={tournamentId() ?? ''}
              disabled={busy() || unused.loading}
              onChange={(e) => {
                const v = Number(e.currentTarget.value)
                setTournamentId(Number.isFinite(v) && v > 0 ? v : null)
                setError(null)
              }}
            >
              <option value="">Select tournament…</option>
              <For each={unused() ?? []}>
                {(t) => (
                  <option value={t.id}>{displayValue(t.name) || t.link}</option>
                )}
              </For>
            </select>
          </span>
        </label>

        <div class="fantasy-create__row">
          <label class="field">
            <span class="field__label">Min elo cost</span>
            <span class="field__shell">
              <input
                class="field__input"
                type="number"
                value={costMin()}
                disabled={busy()}
                onInput={(e) => setCostMin(Number(e.currentTarget.value) || 0)}
              />
            </span>
          </label>
          <label class="field">
            <span class="field__label">Max elo cost</span>
            <span class="field__shell">
              <input
                class="field__input"
                type="number"
                value={costMax()}
                disabled={busy()}
                onInput={(e) => setCostMax(Number(e.currentTarget.value) || 0)}
              />
            </span>
          </label>
          <label class="field">
            <span class="field__label">Max players</span>
            <span class="field__shell">
              <input
                class="field__input"
                type="number"
                min={1}
                value={maxPlayers()}
                disabled={busy()}
                onInput={(e) => setMaxPlayers(Number(e.currentTarget.value) || 1)}
              />
            </span>
          </label>
          <label class="field">
            <span class="field__label">Max cost sum</span>
            <span class="field__shell">
              <input
                class="field__input"
                type="number"
                min={1}
                value={maxCost()}
                disabled={busy()}
                onInput={(e) => setMaxCost(Number(e.currentTarget.value) || 1)}
              />
            </span>
          </label>
        </div>

        <Show when={tournamentId() != null}>
          <Switch>
            <Match when={preview.loading}>
              <p class="status status--idle">Computing costs…</p>
            </Match>
            <Match when={preview.error}>
              <p class="status status--error">{(preview.error as Error).message}</p>
            </Match>
            <Match when={displayRows().length === 0}>
              <p class="status status--idle">No eligible players on roster</p>
            </Match>
            <Match when={true}>
              <div class="roster roster--preview" role="table" aria-label="Cost preview">
                <div class="roster__head" role="row">
                  <span class="roster__cell" role="columnheader">
                    Player
                  </span>
                  <span class="roster__cell roster__elo" role="columnheader">
                    Elo
                  </span>
                  <span class="roster__cell roster__elo" role="columnheader">
                    Cost
                  </span>
                </div>
                <For each={displayRows()}>
                  {(row) => (
                    <div class="roster__row" role="row">
                      <span class="roster__cell roster__player" role="cell">
                        <Player name={row.name} link={row.link} race={row.race} />
                      </span>
                      <span class="roster__cell roster__elo" role="cell">
                        {row.elo.toFixed(0)}
                      </span>
                      <span class="roster__cell" role="cell">
                        <input
                          class="field__input fantasy-cost-input"
                          type="number"
                          min={0}
                          value={row.cost}
                          disabled={busy()}
                          onInput={(e) => {
                            const n = Number(e.currentTarget.value)
                            setOverrides((prev) => ({
                              ...prev,
                              [row.tournamentPlayerId]: Number.isFinite(n) ? Math.max(0, Math.round(n)) : 0,
                            }))
                          }}
                        />
                      </span>
                    </div>
                  )}
                </For>
              </div>
            </Match>
          </Switch>
        </Show>

        <button type="submit" class="btn btn--primary" disabled={busy() || tournamentId() == null}>
          {busy() ? 'Creating…' : 'Create league'}
        </button>
      </form>
      <Show when={error()}>
        <p class="status status--error">{error()}</p>
      </Show>

      <hr class="rule" />

      <Switch>
        <Match when={leagues.loading}>
          <p class="status status--idle">Locking leagues uplink…</p>
        </Match>
        <Match when={leagues.error}>
          <p class="status status--error">{(leagues.error as Error).message}</p>
        </Match>
        <Match when={(leagues() ?? []).length === 0}>
          <p class="status status--idle">No fantasy leagues yet</p>
        </Match>
        <Match when={leagues()}>
          {(rows) => (
            <div class="roster roster--leagues" role="table" aria-label="Fantasy leagues">
              <div class="roster__head" role="row">
                <span class="roster__cell" role="columnheader">
                  League
                </span>
                <span class="roster__cell" role="columnheader">
                  Status
                </span>
                <span class="roster__cell roster__elo" role="columnheader">
                  Caps
                </span>
              </div>
              <For each={rows()}>
                {(l) => {
                  const st = () => leagueStatus(l)
                  return (
                    <button
                      type="button"
                      class="roster__row roster__row--btn"
                      role="row"
                      onClick={() => navigate(`/fantasy-manage/${l.id}`)}
                    >
                      <span class="roster__cell" role="cell">
                        {displayValue(l.tournamentName) || l.tournamentLink}
                      </span>
                      <span class="roster__cell" role="cell">
                        <span
                          classList={{
                            chip: true,
                            'chip--live': st().kind === 'live',
                            'chip--alert': st().kind === 'done',
                            'fantasy-status-chip': true,
                            'fantasy-status-chip--open': st().kind === 'open',
                          }}
                        >
                          {st().label}
                        </span>
                      </span>
                      <span class="roster__cell roster__elo" role="cell">
                        {l.maxPlayers}p / {l.maxCost}c
                      </span>
                    </button>
                  )
                }}
              </For>
            </div>
          )}
        </Match>
      </Switch>
    </ConsoleCard>
  )
}
