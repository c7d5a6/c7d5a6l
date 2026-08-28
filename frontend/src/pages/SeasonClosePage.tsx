import { ConsoleCard } from '../components/ConsoleCard'
import { ChannelHead } from '../components/ChannelChrome'
import { A } from '@solidjs/router'
import {
  For,
  Match,
  Show,
  Switch,
  createEffect,
  createResource,
  createSignal,
  type JSX,
} from 'solid-js'
import { isAdmin } from '../lib/auth'
import {
  closeSeason,
  fetchSeasonClosePreview,
} from '../lib/api/season'
import { displayValue } from '../types/tournament'

function formatSeasonDate(iso: string): string {
  const d = iso.slice(0, 10)
  if (!/^\d{4}-\d{2}-\d{2}$/.test(d)) return iso
  return d
}

/** Admin screen to close the active season and recalculate elos. */
export function SeasonClosePage(): JSX.Element {
  const [preview, { refetch }] = createResource(fetchSeasonClosePreview)
  const [selected, setSelected] = createSignal<Set<number>>(new Set())
  const [initialized, setInitialized] = createSignal(false)
  const [busy, setBusy] = createSignal(false)
  const [msg, setMsg] = createSignal<string | null>(null)
  const [err, setErr] = createSignal<string | null>(null)

  createEffect(() => {
    const data = preview()
    if (!data || initialized()) return
    const ids = new Set<number>()
    for (const t of data.tournaments) {
      if (t.selected) ids.add(t.id)
    }
    setSelected(ids)
    setInitialized(true)
  })

  function toggle(id: number, on: boolean) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (on) next.add(id)
      else next.delete(id)
      return next
    })
  }

  async function onClose() {
    setBusy(true)
    setErr(null)
    setMsg(null)
    try {
      const season = await closeSeason([...selected()])
      setMsg(`Season closed · ${season.name} is now active`)
      setInitialized(false)
      await refetch()
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Close failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <ConsoleCard class="console--wide">
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Admin · Rating Protocol</p>
        <h1 class="brand__title">
          Season <span>Close</span>
        </h1>
      </header>
      <hr class="rule" />

      <div class="channel-stack">
        <ChannelHead tag="Admin" title="Finish season" />

        <Show when={!isAdmin()}>
          <p class="status status--error">Admin access required</p>
        </Show>

        <Switch>
          <Match when={preview.loading}>
            <p class="status status--idle">Locking season uplink…</p>
          </Match>
          <Match when={preview.error}>
            <p class="status status--error">
              {(preview.error as Error)?.message ?? 'Season uplink failed'}
            </p>
          </Match>
          <Match when={preview()}>
            {(data) => (
              <>
                <div class="season-strip">
                  <div class="season-strip__main">
                    <span class="season-strip__name">{data().season.name}</span>
                    <span class="season-strip__meta">
                      Opened {formatSeasonDate(data().season.startedAt)}
                    </span>
                  </div>
                  <Show when={data().season.readyToClose}>
                    <span class="chip chip--alert season-strip__chip">
                      Fantasy finished · ratings pending
                    </span>
                  </Show>
                </div>

                  <Show when={msg()}>
                    <p class="status status--ok">{msg()}</p>
                  </Show>
                  <Show when={err()}>
                    <p class="status status--error">{err()}</p>
                  </Show>

                  <p class="status status--idle season-close__hint">
                    Select tournaments whose matches count toward this season's rating recalculation.
                    The closing fantasy league tournament is always included in an additional pass.
                  </p>

                  <div class="season-close__list" role="group" aria-label="Season tournaments">
                    <Show when={data().tournaments.length === 0}>
                      <p class="status status--idle">No tournaments in the current season window</p>
                    </Show>
                    <For each={data().tournaments}>
                      {(t) => (
                        <label
                          classList={{
                            'season-close__row': true,
                            'season-close__row--fl': t.isFantasySource,
                          }}
                        >
                          <input
                            type="checkbox"
                            class="season-close__check"
                            checked={selected().has(t.id)}
                            disabled={busy()}
                            onChange={(e) => toggle(t.id, e.currentTarget.checked)}
                          />
                          <span class="season-close__label">
                            <span class="season-close__title">
                              {displayValue(t.name) || t.link}
                            </span>
                            <span class="season-close__dates">
                              {t.startDate ?? '—'} → {t.endDate ?? '—'}
                            </span>
                          </span>
                          <span class="season-close__chips">
                            <Show when={t.finished}>
                              <span class="chip chip--live chip--compact">Finished</span>
                            </Show>
                            <Show when={t.isFantasySource}>
                              <span class="chip chip--alert chip--compact">FL pass</span>
                            </Show>
                          </span>
                        </label>
                      )}
                    </For>
                  </div>

                  <div class="actions season-close__actions">
                    <button
                      type="button"
                      class="btn btn--primary"
                      disabled={busy()}
                      onClick={() => void onClose()}
                    >
                      {busy() ? 'Closing season…' : 'Finish season & open next'}
                    </button>
                    <A href="/players" class="btn btn--ghost">
                      Players roster
                    </A>
                  </div>
                </>
            )}
          </Match>
        </Switch>
      </div>
    </ConsoleCard>
  )
}
