import { Show, createSignal, onCleanup, onMount, type JSX } from 'solid-js'
import { fetchAuthConfig, loginWithTelegram } from '../lib/auth'
import type { TelegramAuthPayload } from '../types/user'

export type LoginModalProps = {
  onClose: () => void
}

declare global {
  interface Window {
    __c7d5a6lTelegramOnAuth?: (user: TelegramAuthPayload) => void
  }
}

const AUTH_CB = '__c7d5a6lTelegramOnAuth'

/**
 * Console login modal with official Telegram blue Login Widget.
 * Requires https://c7d5a6l.lo on port 443 (Telegram CSP frame-ancestors).
 */
export function LoginModal(props: LoginModalProps): JSX.Element {
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)
  let widgetHost!: HTMLDivElement

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && !busy()) props.onClose()
  }
  window.addEventListener('keydown', onKey)
  onCleanup(() => {
    window.removeEventListener('keydown', onKey)
    delete window.__c7d5a6lTelegramOnAuth
  })

  onMount(() => {
    window.__c7d5a6lTelegramOnAuth = (user: TelegramAuthPayload) => {
      setError(null)
      setBusy(true)
      void loginWithTelegram(user)
        .then(() => props.onClose())
        .catch((err: unknown) => {
          setError(err instanceof Error ? err.message : 'login failed')
        })
        .finally(() => setBusy(false))
    }

    void (async () => {
      try {
        const { botUsername } = await fetchAuthConfig()
        const username = botUsername.replace(/^@/, '').trim()
        if (!username) throw new Error('bot username missing')

        widgetHost.replaceChildren()
        const script = document.createElement('script')
        script.async = true
        script.src = 'https://telegram.org/js/telegram-widget.js?22'
        script.setAttribute('data-telegram-login', username)
        script.setAttribute('data-size', 'large')
        script.setAttribute('data-radius', '4')
        script.setAttribute('data-onauth', `${AUTH_CB}(user)`)
        script.setAttribute('data-request-access', 'write')
        widgetHost.appendChild(script)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'auth setup failed')
      }
    })()
  })

  return (
    <div
      class="login-modal"
      role="dialog"
      aria-modal="true"
      aria-labelledby="login-modal-title"
      onClick={(e) => {
        if (e.target === e.currentTarget && !busy()) props.onClose()
      }}
    >
      <div class="login-modal__panel motion-drop-in">
        <div class="login-modal__metal" aria-hidden="true" />
        <div class="login-modal__hazard" aria-hidden="true" />
        <div class="login-modal__frame">
          <div class="login-modal__glass">
            <p class="brand__eyebrow atm-phosphor">Uplink · Identity</p>
            <h2 id="login-modal-title" class="login-modal__title">
              Command Login
            </h2>
            <hr class="rule" />
            <p class="login-modal__hint">Sign in with your Telegram account.</p>
            <Show when={error()}>
              <p class="status status--error">{error()}</p>
            </Show>
            <Show when={busy()}>
              <p class="status status--idle">Signing in…</p>
            </Show>
            <div class="login-modal__widget" classList={{ 'login-modal__widget--busy': busy() }} ref={widgetHost} />
            <div class="actions">
              <button type="button" class="btn btn--ghost" disabled={busy()} onClick={() => props.onClose()}>
                Cancel
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
