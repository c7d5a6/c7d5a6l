import { ConsoleCard } from '../components/ConsoleCard'
import { ChannelHead } from '../components/ChannelChrome'
import { TournamentTelemetry } from '../components/TournamentTelemetry'
import { For, Match, Show, Switch, createEffect, createMemo, createResource, createSignal, type JSX } from 'solid-js'
import { A, useNavigate } from '@solidjs/router'
import {
  fetchAdminTournaments,
  fetchTournament,
  ignoreQueueItem,
  parseQueueItem,
  syncTournamentQueue,
} from '../lib/api/tournaments'
import { displayValue, type AdminTournament, type AdminTournamentFlag, type AdminTournamentTab } from '../types/tournament'

const TABS: { id: AdminTournamentTab; label: string }[] = [
  { id: 'queue', label: 'Queue' },
  { id: 'ongoing', label: 'Ongoing' },
  { id: 'parsed', label: 'Parsed' },
  { id: 'finished', label: 'Finished' },
  { id: 'ignored', label: 'Ignored' },
  { id: 'fantasy', label: 'Fantasy' },
  { id: 'all', label: 'All' },
]

const FLAG_LABEL: Record<AdminTournamentFlag, string> = {
  queue: 'Queue',
  ongoing: 'Ongoing',
  parsed: 'Parsed',
  finished: 'Finished',
  ignored: 'Ignored',
  fantasy: 'Fantasy',
}

const PAGE_SIZE = 20

function formatDates(row: AdminTournament): string {
  if (row.startDate && row.endDate && row.startDate !== row.endDate) {
    return `${row.startDate} – ${row.endDate}`
  }
  return displayValue(row.startDate ?? row.endDate)
}

function flagClass(flag: AdminTournamentFlag): string {
  switch (flag) {
    case 'ongoing':
      return 'chip chip--compact chip--live'
    case 'ignored':
      return 'chip chip--compact chip--alert'
    case 'fantasy':
      return 'chip chip--compact chip--fantasy'
    default:
      return 'chip chip--compact'
  }
}

/** Admin tournament queue + stored list. */
export function TournamentsPage(): JSX.Element {
  const navigate = useNavigate()
  const [tab, setTab] = createSignal<AdminTournamentTab>('queue')
  const [page, setPage] = createSignal(1)
  const [busyId, setBusyId] = createSignal<number | null>(null)
  const [syncing, setSyncing] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  createEffect((prev: AdminTournamentTab | undefined) => {
    const next = tab()
    if (prev !== undefined && prev !== next) setPage(1)
    return next
  })

  const listKey = createMemo(() => ({ tab: tab(), page: page() }))
  const [list, { refetch }] = createResource(listKey, (k) =>
    fetchAdminTournaments(k.tab, k.page, PAGE_SIZE),
  )

  const pageCount = createMemo(() => {
    const total = list()?.total ?? 0
    return Math.max(1, Math.ceil(total / PAGE_SIZE))
  })

  async function onSync() {
    setSyncing(true)
    setError(null)
    try {
      await syncTournamentQueue()
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sync failed')
    } finally {
      setSyncing(false)
    }
  }

  async function onParse(row: AdminTournament, e: Event) {
    e.stopPropagation()
    if (row.queueId == null) return
    setBusyId(row.queueId)
    setError(null)
    try {
      const saved = await parseQueueItem(row.queueId)
      if (saved.tournamentId) {
        navigate(`/tournaments/${saved.tournamentId}`)
        return
      }
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Parse failed')
    } finally {
      setBusyId(null)
    }
  }

  async function onIgnore(row: AdminTournament, e: Event) {
    e.stopPropagation()
    if (row.queueId == null) return
    setBusyId(row.queueId)
    setError(null)
    try {
      await ignoreQueueItem(row.queueId)
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ignore failed')
    } finally {
      setBusyId(null)
    }
  }

  function openRow(row: AdminTournament) {
    if (row.tournamentId != null) {
      navigate(`/tournaments/${row.tournamentId}`)
    }
  }

  return (
    <ConsoleCard class="console--wide">
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Leagues · Command Protocol</p>
        <h1 class="brand__title">
          Tours <span>Channel</span>
        </h1>
      </header>
      <hr class="rule" />

      <div class="channel-stack">
        <ChannelHead
          tag="Queue"
          title="Tournament list"
          actions={
            <button
              type="button"
              classList={{ btn: true, 'btn--compact': true, 'btn--busy': syncing() }}
              disabled={syncing() || busyId() != null}
              onClick={() => void onSync()}
            >
              {syncing() ? 'Syncing…' : 'Sync listing'}
            </button>
          }
        />

        <nav class="console__tabs" role="tablist" aria-label="Tournament filters">
          <For each={TABS}>
            {(item) => (
              <button
                type="button"
                role="tab"
                classList={{ tab: true, 'tab--active': tab() === item.id }}
                aria-selected={tab() === item.id}
                onClick={() => setTab(item.id)}
              >
                {item.label}
              </button>
            )}
          </For>
        </nav>

        <Show when={error()}>
          {(err) => <p class="status status--error">{err()}</p>}
        </Show>

        <Switch>
          <Match when={list.loading}>
            <p class="status status--idle">Locking tournament uplink…</p>
          </Match>
          <Match when={list.error}>
            <p class="status status--error">
              {(list.error as Error)?.message ?? 'Tournament uplink failed'}
            </p>
          </Match>
          <Match when={list()}>
            {(data) => (
              <>
                <Show when={data().items.length === 0}>
                  <p class="status status--idle">No tournaments in this filter</p>
                </Show>
                <Show when={data().items.length > 0}>
                  <p class="status status--ok">
                    {data().total} tournament{data().total === 1 ? '' : 's'} · page {data().page} of{' '}
                    {pageCount()}
                  </p>
                  <div class="roster roster--tours" role="table" aria-label="Tournaments">
                    <div class="roster__head" role="row">
                      <span class="roster__cell roster__tour-name" role="columnheader">
                        Tournament
                      </span>
                      <span class="roster__cell roster__tour-dates" role="columnheader">
                        Dates
                      </span>
                      <span class="roster__cell roster__tour-flags" role="columnheader">
                        Status
                      </span>
                      <span class="roster__cell roster__actions" role="columnheader">
                        <span class="points-board__sr-only">Actions</span>
                      </span>
                    </div>
                    <For each={data().items}>
                      {(row) => {
                        const clickable = () => row.tournamentId != null
                        return (
                          <div
                            classList={{
                              roster__row: true,
                              'roster__row--link': clickable(),
                              'roster__row--ignored': row.disabled,
                            }}
                            role="row"
                            tabIndex={clickable() ? 0 : undefined}
                            onClick={() => openRow(row)}
                            onKeyDown={(e) => {
                              if (clickable() && (e.key === 'Enter' || e.key === ' ')) {
                                e.preventDefault()
                                openRow(row)
                              }
                            }}
                          >
                            <span class="roster__cell roster__tour-name" role="cell">
                              <span class="roster__tour-title">{displayValue(row.name)}</span>
                              <a
                                class="roster__tour-link"
                                href={row.link}
                                target="_blank"
                                rel="noreferrer"
                                onClick={(e) => e.stopPropagation()}
                              >
                                Liquipedia
                              </a>
                            </span>
                            <span class="roster__cell roster__tour-dates" role="cell">
                              {formatDates(row)}
                            </span>
                            <span class="roster__cell roster__tour-flags" role="cell">
                              <For each={row.flags}>
                                {(flag) => <span class={flagClass(flag)}>{FLAG_LABEL[flag]}</span>}
                              </For>
                            </span>
                            <span class="roster__cell roster__actions" role="cell">
                              <span class="roster__elo-actions">
                                <Show when={row.queueId != null && !row.flags.includes('parsed')}>
                                  <button
                                    type="button"
                                    classList={{
                                      btn: true,
                                      'btn--compact': true,
                                      'btn--busy': busyId() === row.queueId,
                                    }}
                                    disabled={busyId() != null || syncing()}
                                    onClick={(e) => void onParse(row, e)}
                                  >
                                    Parse
                                  </button>
                                </Show>
                                <Show when={row.queueId != null && !row.disabled}>
                                  <button
                                    type="button"
                                    class="btn btn--ghost btn--compact"
                                    disabled={busyId() != null || syncing()}
                                    onClick={(e) => void onIgnore(row, e)}
                                  >
                                    Ignore
                                  </button>
                                </Show>
                              </span>
                            </span>
                          </div>
                        )
                      }}
                    </For>
                  </div>
                  <div class="pager">
                    <button
                      type="button"
                      class="btn btn--ghost btn--compact"
                      disabled={page() <= 1 || list.loading}
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                    >
                      Prev
                    </button>
                    <span class="pager__status">
                      {page()} / {pageCount()}
                    </span>
                    <button
                      type="button"
                      class="btn btn--ghost btn--compact"
                      disabled={page() >= pageCount() || list.loading}
                      onClick={() => setPage((p) => p + 1)}
                    >
                      Next
                    </button>
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

/** Stored tournament telemetry for a parsed queue row. */
export function TournamentDetailPage(props: { tournamentId: number }): JSX.Element {
  const [data] = createResource(
    () => props.tournamentId,
    (id) => fetchTournament(id),
  )

  return (
    <ConsoleCard class="console--wide">
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Leagues · Command Protocol</p>
        <h1 class="brand__title">
          Tour <span>Detail</span>
        </h1>
      </header>
      <hr class="rule" />
      <p class="status status--idle">
        <A href="/tournaments">← Tournament list</A>
      </p>
      <Switch>
        <Match when={data.loading}>
          <p class="status status--idle">Locking tournament payload…</p>
        </Match>
        <Match when={data.error}>
          <p class="status status--error">
            {(data.error as Error)?.message ?? 'Tournament uplink failed'}
          </p>
        </Match>
        <Match when={data()}>
          {(res) => (
            <TournamentTelemetry
              message={res().message}
              tournament={res().tournament}
              sync={res().tournamentSync}
            />
          )}
        </Match>
      </Switch>
    </ConsoleCard>
  )
}
