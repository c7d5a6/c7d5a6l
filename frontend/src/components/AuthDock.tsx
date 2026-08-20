import { Show, createSignal, type JSX } from 'solid-js'
import { useNavigate } from '@solidjs/router'
import { authUser } from '../lib/auth'
import { ActionPanel } from './ActionPanel'
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
            <ActionPanel
              class="auth-dock__panel"
              label="Login"
              kind="ok"
              title="Login"
              onClick={() => setLoginOpen(true)}
            />
          }
        >
          {(u) => (
            <ActionPanel
              class="auth-dock__panel"
              label={u().alias}
              kind="ok"
              title={u().alias}
              onClick={() => navigate('/me')}
            />
          )}
        </Show>
      </div>
      <Show when={loginOpen()}>
        <LoginModal onClose={() => setLoginOpen(false)} />
      </Show>
    </>
  )
}
