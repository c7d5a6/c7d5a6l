import { ConsoleCard } from '../components/ConsoleCard'
import { ChampionMark } from '../components/ChampionMark'
import { FantasyGroupsPanel } from '../components/FantasyGroupsPanel'
import { FantasyResultsPanel } from '../components/FantasyResultsPanel'
import { FantasyTodayPanel } from '../components/FantasyTodayPanel'
import { TeamScoreMeta } from '../components/FantasyScoreReadout'
import { Player } from '../components/Player'
import { RosterPlayerChip } from '../components/RosterPlayerChip'
import { TeamEditor } from '../components/TeamEditor'
import { UserTitles } from '../components/UserTitles'
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
import { authFetch, authUser } from '../lib/auth'
import {
  fetchActiveFantasyLeague,
  fetchFantasyGroups,
  fetchFantasyMatchBoard,
  fetchFantasyPlayers,
  fetchFantasyTeams,
} from '../lib/api/fantasy'
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

type TabId = 'points' | 'teams' | 'results'
type GroupsShell = 'off' | 'in' | 'out'
type TodayShell = 'off' | 'in' | 'out'

/** Fantasy league channel — points, teams, and results for the active league. */
export function FantasyLeaguePage() {
  const [league] = createResource(fetchActiveFantasyLeague)
  const [tab, setTab] = createSignal<TabId>('points')
  const [editingTeam, setEditingTeam] = createSignal(false)
  const [groupsShell, setGroupsShell] = createSignal<GroupsShell>('off')
  const [todayShell, setTodayShell] = createSignal<TodayShell>('off')

  const leagueId = createMemo(() => league()?.id ?? null)
  const groupsPanelLive = createMemo(() => groupsShell() !== 'off')
  const todayPanelLive = createMemo(() => todayShell() !== 'off')
  const sideLive = createMemo(() => groupsPanelLive() || todayPanelLive())

  const [pointPlayers, { refetch: refetchPlayers }] = createResource(
    () => leagueId(),
    (id) => (id == null ? Promise.resolve([] as FantasyPlayerRow[]) : fetchFantasyPlayers(id, 'elo')),
  )
  const [teams, { refetch: refetchTeams }] = createResource(
    () => leagueId(),
    (id) => (id == null ? Promise.resolve([] as FantasyTeamRow[]) : fetchFantasyTeams(id)),
  )

  const groupsFetchKey = createMemo(() => {
    if (!editingTeam() && !groupsPanelLive()) return null
    return leagueId()
  })
  const [groups] = createResource(groupsFetchKey, (id) => fetchFantasyGroups(id))

  const [matchBoard] = createResource(leagueId, (id) => fetchFantasyMatchBoard(id))

  // Mount side panel only after groups load so height matches content.
  createEffect(() => {
    if (!editingTeam()) return
    if (groups.loading) return
    if (groupsShell() !== 'off') return
    if (groups.state === 'ready' || groups.error) setGroupsShell('in')
  })

  // Match Day stays on for the whole league page; draft Groups takes the side slot while editing.
  createEffect(() => {
    if (editingTeam()) {
      if (todayShell() === 'in') setTodayShell('out')
      return
    }
    if (matchBoard.loading) return
    if (!matchBoard()) {
      if (todayShell() === 'in') setTodayShell('out')
      return
    }
    if (todayShell() !== 'in') setTodayShell('in')
  })

  function setEditing(next: boolean) {
    if (next) {
      setEditingTeam(true)
      if (todayShell() === 'in') setTodayShell('out')
      return
    }
    setEditingTeam(false)
    if (groupsShell() === 'in') setGroupsShell('out')
    else setGroupsShell('off')
  }

  function setResultsTab() {
    setTab('results')
    if (editingTeam()) setEditing(false)
  }

  return (
    <div
      classList={{
        'fantasy-league-layout': true,
        'fantasy-league-layout--side': sideLive(),
      }}
    >
      <Show when={todayPanelLive() && matchBoard()}>
        {(board) => (
          <ConsoleCard
            class="console--side console--side-left"
            hazard="right"
            drop
            exiting={todayShell() === 'out'}
            onExitEnd={() => setTodayShell('off')}
          >
            <FantasyTodayPanel board={board()} teams={teams() ?? []} />
          </ConsoleCard>
        )}
      </Show>

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
                  <a
                    class="hud-link"
                    href={active().tournamentLink}
                    target="_blank"
                    rel="noreferrer"
                    title="Open Liquipedia page"
                  >
                    <span class="hud-link__text">
                      {displayValue(active().tournamentName) || active().tournamentLink}
                    </span>
                    <svg class="hud-link__ext" viewBox="0 0 16 16" aria-hidden="true">
                      <path
                        d="M6 3 H3 V13 H13 V10 M8 3 H13 V8 M13 3 L7 9"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="1.6"
                        stroke-linejoin="miter"
                      />
                    </svg>
                  </a>
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
                    {active().maxPlayers}p / {active().maxCost}c
                  </span>
                </div>

                <FantasyMyTeamDock
                  league={active()}
                  teams={teams() ?? []}
                  players={pointPlayers() ?? []}
                  editing={editingTeam()}
                  onEditingChange={setEditing}
                  onSaved={async () => {
                    await Promise.all([refetchTeams(), refetchPlayers()])
                  }}
                />

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
                  <button
                    type="button"
                    role="tab"
                    classList={{ tab: true, 'tab--active': tab() === 'results' }}
                    aria-selected={tab() === 'results'}
                    onClick={() => setResultsTab()}
                  >
                    Results
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
                      loading={teams.loading}
                      error={teams.error}
                      teams={teams() ?? []}
                    />
                  </Match>
                  <Match when={tab() === 'results'}>
                    <FantasyResultsPanel
                      board={matchBoard()}
                      loading={matchBoard.loading}
                      error={matchBoard.error}
                    />
                  </Match>
                </Switch>
              </>
            )}
          </Match>
        </Switch>
      </ConsoleCard>

      <Show when={groupsPanelLive()}>
        <ConsoleCard
          class="console--side"
          hazard="right"
          drop
          exiting={groupsShell() === 'out'}
          onExitEnd={() => setGroupsShell('off')}
        >
          <FantasyGroupsPanel groups={groups() ?? []} error={groups.error} />
        </ConsoleCard>
      </Show>
    </div>
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
                      <Player
                        name={row.name}
                        link={row.link}
                        race={row.race}
                        loser={row.defeated && !row.isWinner}
                      />
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

function FantasyMyTeamDock(props: {
  league: FantasyLeague
  teams: FantasyTeamRow[]
  players: FantasyPlayerRow[]
  editing: boolean
  onEditingChange: (editing: boolean) => void
  onSaved: () => Promise<void>
}) {
  const me = () => authUser()
  const myTeam = createMemo(() => {
    const u = me()
    if (!u) return null
    return props.teams.find((t) => t.userId === u.id) ?? null
  })
  const canEdit = createMemo(() => !!me() && !props.league.started && !props.league.finished)
  const [selectedIds, setSelectedIds] = createSignal<number[]>([])
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  function startEdit() {
    const t = myTeam()
    setSelectedIds(t ? t.members.map((m) => m.fantasyPlayerId) : [])
    props.onEditingChange(true)
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
      props.onEditingChange(false)
      await props.onSaved()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Show when={canEdit()}>
      <div class="fantasy-my-team fantasy-my-team--dock">
        <Show
          when={props.editing}
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
            <button
              type="button"
              class="btn btn--ghost"
              disabled={busy()}
              onClick={() => props.onEditingChange(false)}
            >
              Cancel
            </button>
          </div>
          <Show when={error()}>
            <p class="status status--error">{error()}</p>
          </Show>
        </Show>
      </div>
    </Show>
  )
}

function FantasyTeamsPanel(props: {
  loading: boolean
  error: unknown
  teams: FantasyTeamRow[]
}) {
  const [hoverPlayerId, setHoverPlayerId] = createSignal<number | null>(null)

  return (
    <div class="telemetry__panel" role="tabpanel">
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
                        <UserTitles titles={team.titles} />
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
                                playerId={m.fantasyPlayerId}
                                name={m.name}
                                link={m.link}
                                race={m.race}
                                cost={m.cost}
                                points={m.pointsEarned}
                                defeated={m.defeated}
                                isWinner={m.isWinner}
                                hoverId={hoverPlayerId}
                                onHighlight={setHoverPlayerId}
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
