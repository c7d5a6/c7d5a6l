import { Show, createSignal, type JSX } from 'solid-js'
import { useNavigate } from '@solidjs/router'
import { authUser } from '../lib/auth'
import { LoginModal } from './LoginModal'

/**
 * Top-right auth dock — Login opens modal; alias navigates to /me.
 */
export function AuthDock(): JSX.Element {
  const navigate = useNavigate()
  const [loginOpen, setLoginOpen] = createSignal(false)

  return (
    <>
      <div class="auth-dock motion-drop-in" aria-label="Account">
        <Show
          when={authUser()}
          fallback={
            <AuthPanel label="Login" onClick={() => setLoginOpen(true)} />
          }
        >
          {(u) => (
            <AuthPanel label={u().alias} onClick={() => navigate('/me')} />
          )}
        </Show>
      </div>
      <Show when={loginOpen()}>
        <LoginModal onClose={() => setLoginOpen(false)} />
      </Show>
    </>
  )
}

function AuthPanel(props: { label: string; onClick: () => void }) {
  return (
    <div class="action-panel action-panel--ok auth-dock__panel">
      <div class="action-panel__metal action-panel__metal--outer" aria-hidden="true" />
      <div class="action-panel__hazard" aria-hidden="true" />
      <div class="action-panel__metal action-panel__metal--mid" aria-hidden="true" />
      <div class="action-panel__well">
        <div class="action-panel__red">
          <button
            type="button"
            class="action-panel__btn action-panel__btn--ok"
            title={props.label}
            onClick={() => props.onClick()}
          >
            {props.label}
          </button>
        </div>
      </div>
    </div>
  )
}
