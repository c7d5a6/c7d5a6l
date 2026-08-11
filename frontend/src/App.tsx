import { createMemo, createSignal, Show } from 'solid-js'
import { ConsoleCard } from './components/ConsoleCard'
import { TournamentTelemetry } from './components/TournamentTelemetry'
import { validateLiquipediaURL } from './lib/liquipedia'
import type { ErrorResponse, ParseResponse } from './types/tournament'

function App() {
  const [url, setUrl] = createSignal('')
  const [touched, setTouched] = createSignal(false)
  const [clientError, setClientError] = createSignal<string | null>(null)
  const [serverError, setServerError] = createSignal<string | null>(null)
  const [result, setResult] = createSignal<ParseResponse | null>(null)
  const [submitting, setSubmitting] = createSignal(false)

  const live = createMemo(() => {
    const value = url().trim()
    if (!value) return { state: 'empty' as const }
    const validated = validateLiquipediaURL(value)
    if (validated.ok) return { state: 'ok' as const, url: validated.url }
    return { state: 'bad' as const, error: validated.error }
  })

  const canSubmit = createMemo(() => live().state === 'ok' && !submitting())

  const uplinkFault = createMemo(
    () =>
      Boolean(clientError()) ||
      Boolean(serverError()) ||
      (touched() && live().state === 'bad'),
  )

  function clearForm() {
    setUrl('')
    setTouched(false)
    setClientError(null)
    setServerError(null)
    setResult(null)
  }

  async function onSubmit(e: Event) {
    e.preventDefault()
    setTouched(true)
    setClientError(null)
    setServerError(null)
    setResult(null)

    const validated = validateLiquipediaURL(url())
    if (!validated.ok) {
      setClientError(validated.error)
      return
    }

    setSubmitting(true)
    try {
      const res = await fetch('/api/parse', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
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

  return (
    <div class="stage">
      <div class="stage__art" aria-hidden="true" />
      <div class="stage__grid" aria-hidden="true" />
      <div class="stage__crt" aria-hidden="true" />
      <div class="stage__scan" aria-hidden="true" />
      <div class="stage__vignette" aria-hidden="true" />

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
              <TournamentTelemetry
                message={res().message}
                tournament={res().tournament}
              />
            </>
          )}
        </Show>
      </ConsoleCard>
    </div>
  )
}

export default App
