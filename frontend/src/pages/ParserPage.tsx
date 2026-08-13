import { createEffect, createMemo, createSignal, Match, Show, Switch } from 'solid-js'
import { Portal } from 'solid-js/web'
import { ActionDock } from '../components/ActionDock'
import { ConsoleCard } from '../components/ConsoleCard'
import { PlayerTelemetry } from '../components/PlayerTelemetry'
import { TournamentTelemetry } from '../components/TournamentTelemetry'
import { authFetch } from '../lib/auth'
import { validateLiquipediaURL } from '../lib/liquipedia'
import type {
  ErrorResponse,
  ParseResponse,
  SavePlayerResponse,
  SaveTournamentResponse,
} from '../types/tournament'

type DockHeld = {
  kind: 'player' | 'tournament'
  dockAction: 'add' | 'update'
}

/** ASL uplink / parse console + optional save dock (ported outside slide). */
export function ParserPage() {
  const [url, setUrl] = createSignal('')
  const [touched, setTouched] = createSignal(false)
  const [clientError, setClientError] = createSignal<string | null>(null)
  const [serverError, setServerError] = createSignal<string | null>(null)
  const [result, setResult] = createSignal<ParseResponse | null>(null)
  const [submitting, setSubmitting] = createSignal(false)
  const [dockBusy, setDockBusy] = createSignal(false)
  const [dockExiting, setDockExiting] = createSignal(false)
  const [dockHeld, setDockHeld] = createSignal<DockHeld | null>(null)

  const live = createMemo(() => {
    const value = url().trim()
    if (!value) return { state: 'empty' as const }
    const validated = validateLiquipediaURL(value)
    if (validated.ok) return { state: 'ok' as const, url: validated.url }
    return { state: 'bad' as const, error: validated.error }
  })

  const canSubmit = createMemo(() => live().state === 'ok' && !submitting() && !dockBusy())

  const uplinkFault = createMemo(
    () =>
      Boolean(clientError()) ||
      Boolean(serverError()) ||
      (touched() && live().state === 'bad'),
  )

  const activeDock = createMemo((): DockHeld | null => {
    const res = result()
    if (!res) return null
    if (res.pageType === 'player' && res.player && res.playerSync) {
      const sync = res.playerSync
      const needsAdd = !sync.exists
      const needsUpdate = sync.exists && !sync.same
      if (!needsAdd && !needsUpdate) return null
      return { kind: 'player', dockAction: needsUpdate ? 'update' : 'add' }
    }
    if (res.pageType === 'tournament' && res.tournament && res.tournamentSync) {
      const sync = res.tournamentSync
      const needsAdd = !sync.exists
      const needsUpdate = sync.exists && !sync.same
      if (!needsAdd && !needsUpdate) return null
      return { kind: 'tournament', dockAction: needsUpdate ? 'update' : 'add' }
    }
    return null
  })

  createEffect(() => {
    const dock = activeDock()
    if (dock) {
      setDockHeld(dock)
      return
    }
    if (dockHeld() && !dockExiting()) {
      setDockExiting(true)
    }
  })

  function clearForm() {
    setUrl('')
    setTouched(false)
    setClientError(null)
    setServerError(null)
    setResult(null)
    setDockExiting(false)
    setDockHeld(null)
  }

  function dismissDockAction() {
    if (dockExiting()) return
    setResult(null)
    setServerError(null)
    setDockExiting(true)
  }

  function finishDockExit() {
    if (!dockExiting()) return
    setDockExiting(false)
    setDockHeld(null)
  }

  async function saveFromDock() {
    const res = result()
    const held = dockHeld()
    if (!res || !held) return

    setDockBusy(true)
    setServerError(null)
    try {
      if (held.kind === 'player') {
        if (!res.player) return
        const response = await authFetch('/api/players', {
          method: 'POST',
          body: JSON.stringify(res.player),
        })
        const body = (await response.json()) as SavePlayerResponse | ErrorResponse
        if (!response.ok) {
          setServerError('error' in body ? body.error : `Save failed (${response.status})`)
          return
        }
        const saved = body as SavePlayerResponse
        setResult({
          message: saved.message,
          pageType: 'player',
          player: saved.player,
          playerSync: saved.playerSync,
        })
        return
      }

      if (!res.tournament) return
      const response = await authFetch('/api/tournaments', {
        method: 'POST',
        body: JSON.stringify(res.tournament),
      })
      const body = (await response.json()) as SaveTournamentResponse | ErrorResponse
      if (!response.ok) {
        setServerError('error' in body ? body.error : `Save failed (${response.status})`)
        return
      }
      const saved = body as SaveTournamentResponse
      setResult({
        message: saved.message,
        pageType: 'tournament',
        tournament: saved.tournament,
        tournamentSync: saved.tournamentSync,
      })
    } catch {
      setServerError('Uplink offline — start backend on :18765')
    } finally {
      setDockBusy(false)
    }
  }

  async function onSubmit(e: Event) {
    e.preventDefault()
    setTouched(true)
    setClientError(null)
    setServerError(null)
    setResult(null)
    setDockExiting(false)
    setDockHeld(null)

    const validated = validateLiquipediaURL(url())
    if (!validated.ok) {
      setClientError(validated.error)
      return
    }

    setSubmitting(true)
    try {
      const res = await authFetch('/api/parse', {
        method: 'POST',
        body: JSON.stringify({ url: validated.url }),
      })

      const body = (await res.json()) as ParseResponse | ErrorResponse
      if (!res.ok) {
        setServerError('error' in body ? body.error : `Request failed (${res.status})`)
        return
      }

      setResult(body as ParseResponse)
    } catch {
      setServerError('Uplink offline — start backend on :18765')
    } finally {
      setSubmitting(false)
    }
  }

  const dockVisible = createMemo(() => Boolean(dockHeld()))

  return (
    <>
      <ConsoleCard>
        <header class="brand">
          <p class="brand__eyebrow atm-phosphor">Brood War · Command Protocol</p>
          <h1 class="brand__title">
            ASL <span>Uplink</span>
          </h1>
        </header>

        <div class="meta-row">
          <span
            classList={{
              chip: true,
              'chip--live': !uplinkFault(),
              'chip--alert': uplinkFault(),
            }}
          >
            {uplinkFault() ? 'Channel fault' : 'Channel open'}
          </span>
          <span class="chip">
            Target{' '}
            <a href="https://liquipedia.net/" target="_blank" rel="noreferrer">
              liquipedia.net
            </a>
          </span>
        </div>

        <hr class="rule" />

        <form class="field" onSubmit={onSubmit}>
          <label class="field__label" for="uplink-url">
            <span>Link uplink</span>
            <span class="field__hint">https required</span>
          </label>

          <div
            classList={{
              field__shell: true,
              'field__shell--ok': touched() && live().state === 'ok',
              'field__shell--bad':
                touched() && (live().state === 'bad' || Boolean(clientError())),
            }}
          >
            <span class="field__glyph" aria-hidden="true" />
            <input
              id="uplink-url"
              class="field__input"
              type="url"
              name="url"
              value={url()}
              onInput={(e) => {
                setUrl(e.currentTarget.value)
                setTouched(true)
                setClientError(null)
                setServerError(null)
                setResult(null)
                setDockExiting(false)
                setDockHeld(null)
              }}
              onBlur={() => setTouched(true)}
              placeholder="https://liquipedia.net/…"
              autocomplete="off"
              spellcheck={false}
              aria-invalid={touched() && live().state === 'bad' ? true : undefined}
              aria-describedby="uplink-status"
            />
          </div>

          <div class="actions">
            <button
              type="submit"
              classList={{ btn: true, 'btn--busy': submitting() }}
              disabled={!canSubmit()}
            >
              {submitting() ? 'Transmitting…' : 'Engage parse'}
            </button>
            <button
              type="button"
              class="btn btn--ghost"
              onClick={clearForm}
              disabled={!url() && !result() && !clientError() && !serverError()}
            >
              Clear
            </button>
          </div>
        </form>

        <div id="uplink-status" aria-live="polite">
          <Show
            when={
              clientError() ??
              (touched() && live().state === 'bad' ? live().error : null)
            }
          >
            {(err) => <p class="status status--error">{err()}</p>}
          </Show>
          <Show when={serverError()}>
            {(err) => <p class="status status--error">{err()}</p>}
          </Show>
          <Show when={!clientError() && !serverError() && live().state === 'ok' && !result()}>
            <p class="status status--ok">Link verified — ready to transmit</p>
          </Show>
          <Show when={!touched() && !result() && !serverError()}>
            <p class="status status--idle">Awaiting valid liquipedia.net URL</p>
          </Show>
        </div>

        <Show when={result()}>
          {(res) => (
            <>
              <hr class="rule" />
              <Switch>
                <Match
                  when={
                    res().pageType === 'tournament' && res().tournament && res().tournamentSync
                      ? res()
                      : undefined
                  }
                >
                  {(data) => (
                    <TournamentTelemetry
                      message={data().message}
                      tournament={data().tournament!}
                      sync={data().tournamentSync!}
                    />
                  )}
                </Match>
                <Match
                  when={
                    res().pageType === 'player' && res().player && res().playerSync
                      ? res()
                      : undefined
                  }
                >
                  {(data) => (
                    <PlayerTelemetry
                      message={data().message}
                      player={data().player!}
                      sync={data().playerSync!}
                    />
                  )}
                </Match>
              </Switch>
            </>
          )}
        </Show>
      </ConsoleCard>

      <Portal>
        <Show when={dockVisible() ? dockHeld() : null}>
          {(data) => (
            <ActionDock
              okLabel={data().dockAction === 'update' ? 'Update' : 'Add'}
              busy={dockBusy()}
              exiting={dockExiting()}
              onOk={saveFromDock}
              onCancel={dismissDockAction}
              onExitEnd={finishDockExit}
            />
          )}
        </Show>
      </Portal>
    </>
  )
}
