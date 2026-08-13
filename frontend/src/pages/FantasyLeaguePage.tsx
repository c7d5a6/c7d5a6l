import { ConsoleCard } from '../components/ConsoleCard'
import { ChampionMark } from '../components/ChampionMark'
import { TeamScoreMeta } from '../components/FantasyScoreReadout'
import { Player } from '../components/Player'
import { RosterPlayerChip } from '../components/RosterPlayerChip'
import { TeamEditor } from '../components/TeamEditor'
import { For, Match, Show, Switch, createMemo, createResource, createSignal } from 'solid-js'
import { authFetch, authUser } from '../lib/auth'
import { displayValue } from '../types/tournament'
import {
  POINT_STAGES,
  sortByElo,
  stagePoints,
  stageReached,
  type FantasyLeague,
  type FantasyPlayerRow,
  type FantasyTeamRow,
  type PointStage,
} from '../types/fantasy'

type TabId = 'points' | 'teams'

async function fetchActiveLeague(): Promise<FantasyLeague | null> {
  const res = await authFetch('/api/fantasy-leagues/active')
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`fantasy league uplink failed (${res.status})`)
  const data = (await res.json()) as { league: FantasyLeague }
  return data.league ?? null
}

async function fetchPlayers(leagueId: number, sort: string): Promise<FantasyPlayerRow[]> {
  const res = await authFetch(`/api/fantasy-leagues/${leagueId}/players?sort=${sort}`)
  if (!res.ok) throw new Error(`fantasy players uplink failed (${res.status})`)
  const data = (await res.json()) as { players: FantasyPlayerRow[] }
  return data.players ?? []
}

async function fetchTeams(leagueId: number): Promise<FantasyTeamRow[]> {
  const res = await authFetch(`/api/fantasy-leagues/${leagueId}/teams`)
  if (!res.ok) throw new Error(`fantasy teams uplink failed (${res.status})`)
  const data = (await res.json()) as { teams: FantasyTeamRow[] }
  return data.teams ?? []
}

/** Fantasy league channel — points and teams for the active league. */
export function FantasyLeaguePage() {
  const [league] = createResource(fetchActiveLeague)
  const [tab, setTab] = createSignal<TabId>('points')

  const leagueId = createMemo(() => league()?.id ?? null)

  const [pointPlayers, { refetch: refetchPlayers }] = createResource(
    () => leagueId(),
    (id) => (id == null ? Promise.resolve([] as FantasyPlayerRow[]) : fetchPlayers(id, 'elo')),
  )
  const [teams, { refetch: refetchTeams }] = createResource(
    () => leagueId(),
    (id) => (id == null ? Promise.resolve([] as FantasyTeamRow[]) : fetchTeams(id)),
  )

  return (
    <ConsoleCard class="console--wide">
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">League · Command Protocol</p>
        <h1 class="brand__title">
          Fantasy <span>League</span>
        </h1>
      </header>
      <hr class="rule" />

      <Switch>
        <Match when={league.loading}>
          <p class="status status--idle">Locking fantasy uplink…</p>
        </Match>
        <Match when={league.error}>
          <p class="status status--error">
            {(league.error as Error)?.message ?? 'Fantasy uplink failed'}
          </p>
        </Match>
        <Match when={league() === null}>
          <p class="status status--idle">No fantasy league in database</p>
        </Match>
        <Match when={league()}>
          {(active) => (
            <>
              <div class="fantasy-league-banner">
                <Show when={active().finished}>
                  <span class="chip chip--alert">Finished</span>
                </Show>
                <Show when={active().started && !active().finished}>
                  <span class="chip chip--live">Started</span>
                </Show>
                <Show when={!active().started && !active().finished}>
                  <span class="chip fantasy-status-chip--open">Open</span>
                </Show>
                <span class="fantasy-league-banner__caps">
                  {displayValue(active().tournamentName)} · {active().maxPlayers}p / {active().maxCost}c
                </span>
              </div>

              <nav class="console__tabs" role="tablist" aria-label="Fantasy sections">
                <button
                  type="button"
                  role="tab"
                  classList={{ tab: true, 'tab--active': tab() === 'points' }}
                  aria-selected={tab() === 'points'}
                  onClick={() => setTab('points')}
                >
                  Points
                </button>
                <button
                  type="button"
                  role="tab"
                  classList={{ tab: true, 'tab--active': tab() === 'teams' }}
                  aria-selected={tab() === 'teams'}
                  onClick={() => setTab('teams')}
                >
                  Teams
                </button>
              </nav>

              <Switch>
                <Match when={tab() === 'points'}>
                  <FantasyPointsBoard
                    loading={pointPlayers.loading}
                    error={pointPlayers.error}
                    rows={pointPlayers() ?? []}
                  />
                </Match>
                <Match when={tab() === 'teams'}>
                  <FantasyTeamsPanel
                    league={active()}
                    loading={teams.loading}
                    error={teams.error}
                    teams={teams() ?? []}
                    players={pointPlayers() ?? []}
                    onSaved={async () => {
                      await Promise.all([refetchTeams(), refetchPlayers()])
                    }}
                  />
                </Match>
              </Switch>
            </>
          )}
        </Match>
      </Switch>
    </ConsoleCard>
  )
}

function FantasyPointsBoard(props: {
  loading: boolean
  error: unknown
  rows: FantasyPlayerRow[]
}) {
  const rows = createMemo(() => sortByElo(props.rows))

  return (
    <div class="telemetry__panel" role="tabpanel">
      <Switch>
        <Match when={props.loading}>
          <p class="status status--idle">Loading points board…</p>
        </Match>
        <Match when={props.error}>
          <p class="status status--error">
            {(props.error as Error)?.message ?? 'Failed to load points'}
          </p>
        </Match>
        <Match when={props.rows.length === 0}>
          <p class="status status--idle">No fantasy players</p>
        </Match>
        <Match when={true}>
          <div class="points-board" role="table" aria-label="Points earned by bracket stage">
            <div class="points-board__head" role="row">
              <span class="points-board__cell points-board__player" role="columnheader">
                Player
              </span>
              <span class="points-board__cell points-board__total" role="columnheader">
                Points
                <span class="points-board__sub">(total)</span>
              </span>
              <span class="points-board__cell points-board__stages" role="columnheader">
                <span class="points-board__stages-labels" aria-hidden="true">
                  <For each={[...POINT_STAGES]}>{(s) => <span>{s}</span>}</For>
                </span>
              </span>
              <span class="points-board__cell points-board__cost" role="columnheader">
                Cost
              </span>
              <span class="points-board__cell points-board__diff" role="columnheader">
                Diff
              </span>
            </div>
            <For each={rows()}>
              {(row) => {
                const diff = () => row.pointsEarned - row.cost
                return (
                  <div
                    classList={{
                      'points-board__row': true,
                      'points-board__row--defeated': row.defeated,
                      'points-board__row--winner': row.isWinner,
                    }}
                    role="row"
                  >
                    <span class="points-board__cell points-board__player" role="cell">
                      <Show when={row.isWinner}>
                        <ChampionMark />
                      </Show>
                      <Player name={row.name} link={row.link} race={row.race} />
                    </span>
                    <span class="points-board__cell points-board__total" role="cell">
                      <span class="points-board__total-val">{row.pointsEarned}</span>
                    </span>
                    <span class="points-board__cell points-board__stages" role="cell">
                      <StageProgress row={row} />
                    </span>
                    <span class="points-board__cell points-board__cost" role="cell">
                      {row.cost}
                    </span>
                    <span
                      classList={{
                        'points-board__cell': true,
                        'points-board__diff': true,
                        'points-board__diff--up': diff() > 0,
                        'points-board__diff--down': diff() < 0,
                        'points-board__diff--flat': diff() === 0,
                      }}
                      role="cell"
                    >
                      {diff() > 0 ? `+${diff()}` : diff()}
                    </span>
                  </div>
                )
              }}
            </For>
          </div>
        </Match>
      </Switch>
    </div>
  )
}

function StageProgress(props: { row: FantasyPlayerRow }) {
  const reached = () => stageReached(props.row)
  const pct = () => Math.max(0, Math.min(100, (reached() / POINT_STAGES.length) * 100))

  return (
    <div
      classList={{
        'stage-rail': true,
        'stage-rail--defeated': props.row.defeated,
        'stage-rail--winner': props.row.isWinner,
      }}
      role="img"
      aria-label={`Reached ${POINT_STAGES[Math.max(0, reached() - 1)] ?? 'none'} (${reached()}/${POINT_STAGES.length})`}
    >
          <div class="stage-rail__nums" aria-hidden="true">
        <For each={[...POINT_STAGES]}>
          {(stage) => {
            const v = () => stagePoints(props.row, stage as PointStage)
            return (
              <span
                classList={{
                  'stage-rail__num': true,
                  'stage-rail__num--empty': v() == null,
                }}
              >
                {v() == null ? '·' : v()}
              </span>
            )
          }}
        </For>
      </div>
      <div class="stage-rail__track">
        <div class="stage-rail__fill" style={{ width: `${pct()}%` }} />
        <div class="stage-rail__segments">
          <For each={[...POINT_STAGES]}>
            {(_stage, i) => {
              const on = () => i() < reached()
              const tip = () => i() === reached() - 1
              return (
                <span
                  classList={{
                    'stage-rail__seg': true,
                    'stage-rail__seg--on': on(),
                    'stage-rail__seg--tip': tip(),
                  }}
                />
              )
            }}
          </For>
        </div>
      </div>
    </div>
  )
}

function FantasyTeamsPanel(props: {
  league: FantasyLeague
  loading: boolean
  error: unknown
  teams: FantasyTeamRow[]
  players: FantasyPlayerRow[]
  onSaved: () => Promise<void>
}) {
  const me = () => authUser()
  const myTeam = createMemo(() => {
    const u = me()
    if (!u) return null
    return props.teams.find((t) => t.userId === u.id) ?? null
  })
  const canEdit = createMemo(() => !!me() && !props.league.started && !props.league.finished)
  const [editing, setEditing] = createSignal(false)
  const [selectedIds, setSelectedIds] = createSignal<number[]>([])
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  function startEdit() {
    const t = myTeam()
    setSelectedIds(t ? t.members.map((m) => m.fantasyPlayerId) : [])
    setEditing(true)
    setError(null)
  }

  async function saveTeam() {
    setBusy(true)
    setError(null)
    try {
      const res = await authFetch(`/api/fantasy-leagues/${props.league.id}/my-team`, {
        method: 'PUT',
        body: JSON.stringify({ fantasyPlayerIds: selectedIds() }),
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
      setEditing(false)
      await props.onSaved()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div class="telemetry__panel" role="tabpanel">
      <Show when={canEdit()}>
        <div class="fantasy-my-team">
          <Show
            when={editing()}
            fallback={
              <button type="button" class="btn btn--primary" onClick={startEdit}>
                {myTeam() ? 'Edit my team' : 'Create my team'}
              </button>
            }
          >
            <TeamEditor
              players={sortByElo(props.players)}
              selectedIds={selectedIds()}
              maxPlayers={props.league.maxPlayers}
              maxCost={props.league.maxCost}
              disabled={busy()}
              onChange={setSelectedIds}
            />
            <div class="actions">
              <button type="button" class="btn btn--primary" disabled={busy()} onClick={() => void saveTeam()}>
                {busy() ? 'Saving…' : 'Save team'}
              </button>
              <button type="button" class="btn btn--ghost" disabled={busy()} onClick={() => setEditing(false)}>
                Cancel
              </button>
            </div>
            <Show when={error()}>
              <p class="status status--error">{error()}</p>
            </Show>
          </Show>
        </div>
      </Show>

      <Switch>
        <Match when={props.loading}>
          <p class="status status--idle">Loading teams…</p>
        </Match>
        <Match when={props.error}>
          <p class="status status--error">
            {(props.error as Error)?.message ?? 'Failed to load teams'}
          </p>
        </Match>
        <Match when={props.teams.length === 0}>
          <p class="status status--idle">No fantasy teams yet</p>
        </Match>
        <Match when={true}>
          <div class="fantasy-teams">
            <For each={props.teams}>
              {(team) => {
                const champion = () => team.members.find((m) => m.isWinner) ?? null
                const rosterActive = () =>
                  team.members.length > 0 && team.members.some((m) => !m.defeated)
                return (
                  <section
                    classList={{
                      'fantasy-team': true,
                      'fantasy-team--ranked': true,
                      'fantasy-team--has-champion': !!champion(),
                      'fantasy-team--active': rosterActive(),
                    }}
                  >
                    <header class="fantasy-team__head">
                      <div class="fantasy-team__identity">
                        <span
                          classList={{
                            'fantasy-team__rank': true,
                            'fantasy-team__rank--top': team.rank === 1,
                          }}
                          data-rank={team.rank}
                        >
                          <span class="fantasy-team__rank-hash">#</span>
                          {team.rank}
                        </span>
                        <h2 class="fantasy-team__title">{team.userAlias}</h2>
                        <Show when={rosterActive()}>
                          <span
                            class="chip chip--live fantasy-team__live"
                            title="Roster still active"
                            aria-label="Roster still active"
                          />
                        </Show>
                      </div>
                      <TeamScoreMeta points={team.points} cost={team.cost} />
                    </header>
                    <div class="fantasy-team__bay">
                      <ul class="fantasy-team__list">
                        <For each={team.members} fallback={<li class="status status--idle">Empty roster</li>}>
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
                  </section>
                )
              }}
            </For>
          </div>
        </Match>
      </Switch>
    </div>
  )
}
