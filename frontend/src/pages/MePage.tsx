import { ConsoleCard } from '../components/ConsoleCard'
import { ChannelHead, NestedPlate } from '../components/ChannelChrome'
import { UserTitles } from '../components/UserTitles'
import { Show, createEffect, createSignal, type JSX } from 'solid-js'
import { useNavigate } from '@solidjs/router'
import { authUser, homePath, logout, updateAlias } from '../lib/auth'
import { APP_VERSION } from '../lib/version'

/** Profile channel — Telegram identity + logout. */
export function MePage(): JSX.Element {
  const navigate = useNavigate()
  const [busy, setBusy] = createSignal(false)
  const [aliasDraft, setAliasDraft] = createSignal('')
  const [aliasDirty, setAliasDirty] = createSignal(false)
  const [aliasBusy, setAliasBusy] = createSignal(false)
  const [aliasError, setAliasError] = createSignal<string | null>(null)
  const [aliasOk, setAliasOk] = createSignal(false)

  createEffect(() => {
    const u = authUser()
    if (u && !aliasDirty()) setAliasDraft(u.alias)
  })

  async function onLogout() {
    setBusy(true)
    try {
      await logout()
      navigate(homePath())
    } finally {
      setBusy(false)
    }
  }

  async function onSaveAlias(e: Event) {
    e.preventDefault()
    setAliasBusy(true)
    setAliasError(null)
    setAliasOk(false)
    try {
      await updateAlias(aliasDraft())
      setAliasDirty(false)
      setAliasOk(true)
    } catch (err) {
      setAliasError(err instanceof Error ? err.message : 'Alias update failed')
    } finally {
      setAliasBusy(false)
    }
  }

  return (
    <ConsoleCard>
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Identity · Command Protocol</p>
        <h1 class="brand__title">
          Operator <span>Profile</span>
        </h1>
      </header>
      <hr class="rule" />

      <div class="channel-stack">
        <ChannelHead tag="Identity" title="Profile" />
      <Show
        when={authUser()}
        fallback={<p class="status status--idle">No active session — use Login on the auth dock.</p>}
      >
        {(u) => (
          <NestedPlate class="me-profile">
            <Show when={u().photoUrl}>
              {(src) => (
                <img class="me-profile__photo" src={src()} alt="" width={96} height={96} referrerPolicy="no-referrer" />
              )}
            </Show>
            <dl class="me-profile__meta">
              <div>
                <dt>Alias</dt>
                <dd>
                  <form class="me-profile__alias" onSubmit={(e) => void onSaveAlias(e)}>
                    <label class="field me-profile__alias-field">
                      <span class="field__shell">
                        <input
                          class="field__input"
                          type="text"
                          value={aliasDraft()}
                          maxlength={64}
                          autocomplete="nickname"
                          disabled={aliasBusy()}
                          onInput={(e) => {
                            setAliasDirty(true)
                            setAliasDraft(e.currentTarget.value)
                            setAliasOk(false)
                            setAliasError(null)
                          }}
                        />
                      </span>
                    </label>
                    <button type="submit" class="btn btn--primary" disabled={aliasBusy() || !aliasDraft().trim()}>
                      {aliasBusy() ? 'Saving…' : 'Save'}
                    </button>
                  </form>
                  <Show when={aliasError()}>
                    <p class="status status--error">{aliasError()}</p>
                  </Show>
                  <Show when={aliasOk()}>
                    <p class="status status--ok">Alias updated</p>
                  </Show>
                </dd>
              </div>
              <div class="me-profile__titles-row">
                <dt>Titles</dt>
                <dd class="me-profile__titles">
                  <Show when={(u().titles ?? []).length > 0} fallback="—">
                    <UserTitles titles={u().titles} />
                  </Show>
                </dd>
              </div>
              <div>
                <dt>Telegram</dt>
                <dd>
                  {u().telegramUsername
                    ? `@${u().telegramUsername}`
                    : u().telegramId != null
                      ? `id ${u().telegramId}`
                      : 'Not linked'}
                </dd>
              </div>
              <div>
                <dt>Name</dt>
                <dd>
                  {u().firstName}
                  {u().lastName ? ` ${u().lastName}` : ''}
                </dd>
              </div>
              <div>
                <dt>Role</dt>
                <dd>{u().role}</dd>
              </div>
            </dl>
            <div class="actions">
              <button type="button" class="btn btn--ghost" disabled={busy()} onClick={() => void onLogout()}>
                {busy() ? 'Signing out…' : 'Logout'}
              </button>
            </div>
          </NestedPlate>
        )}
      </Show>
        <p class="me-profile__version atm-phosphor" aria-label={`App version ${APP_VERSION}`}>
          v{APP_VERSION}
        </p>
      </div>
    </ConsoleCard>
  )
}
