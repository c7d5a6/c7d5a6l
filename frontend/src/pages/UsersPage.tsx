import { ConsoleCard } from '../components/ConsoleCard'
import { For, Match, Show, Switch, createResource, createSignal, type JSX } from 'solid-js'
import { authFetch } from '../lib/auth'
import { readApiError } from '../lib/api/http'
import { fetchUsers } from '../lib/api/users'
import type { AuthUser } from '../types/user'

async function createUser(alias: string): Promise<AuthUser> {
  const res = await authFetch('/api/users', {
    method: 'POST',
    body: JSON.stringify({ alias }),
  })
  if (!res.ok) throw new Error(await readApiError(res, `create user failed (${res.status})`))
  const data = (await res.json()) as { user: AuthUser }
  return data.user
}

function telegramConnected(u: AuthUser): boolean {
  return u.telegramId != null
}

function telegramLabel(u: AuthUser): string {
  if (!telegramConnected(u)) return '—'
  if (u.telegramUsername) return `@${u.telegramUsername}`
  return `id ${u.telegramId}`
}

/** Admin roster — all accounts (alias, telegram link, role). */
export function UsersPage(): JSX.Element {
  const [users, { refetch }] = createResource(fetchUsers)
  const [aliasDraft, setAliasDraft] = createSignal('')
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  async function onCreate(e: Event) {
    e.preventDefault()
    const alias = aliasDraft().trim()
    if (!alias) return
    setBusy(true)
    setError(null)
    try {
      await createUser(alias)
      setAliasDraft('')
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <ConsoleCard>
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Operators · Command Protocol</p>
        <h1 class="brand__title">
          Users <span>Channel</span>
        </h1>
      </header>
      <hr class="rule" />

      <form class="users-create" onSubmit={(e) => void onCreate(e)}>
        <label class="field users-create__field">
          <span class="field__label">New alias</span>
          <span class="field__shell">
            <input
              class="field__input"
              type="text"
              value={aliasDraft()}
              maxlength={64}
              placeholder="Operator callsign"
              autocomplete="off"
              disabled={busy()}
              onInput={(e) => {
                setAliasDraft(e.currentTarget.value)
                setError(null)
              }}
            />
          </span>
        </label>
        <button type="submit" class="btn btn--primary" disabled={busy() || !aliasDraft().trim()}>
          {busy() ? 'Creating…' : 'Create user'}
        </button>
      </form>
      <Show when={error()}>
        <p class="status status--error">{error()}</p>
      </Show>

      <Switch>
        <Match when={users.loading}>
          <p class="status status--idle">Locking users uplink…</p>
        </Match>
        <Match when={users.error}>
          <p class="status status--error">
            {(users.error as Error)?.message ?? 'Users uplink failed'}
          </p>
        </Match>
        <Match when={(users() ?? []).length === 0}>
          <p class="status status--idle">No users in database</p>
        </Match>
        <Match when={users()}>
          {(rows) => (
            <>
              <p class="status status--ok">
                {rows().length} operator{rows().length === 1 ? '' : 's'}
              </p>
              <div class="roster roster--users" role="table" aria-label="Users">
                <div class="roster__head" role="row">
                  <span class="roster__cell roster__alias" role="columnheader">
                    Alias
                  </span>
                  <span class="roster__cell roster__tg-link" role="columnheader">
                    Telegram
                  </span>
                  <span class="roster__cell roster__telegram" role="columnheader">
                    Handle
                  </span>
                  <span class="roster__cell roster__role" role="columnheader">
                    Role
                  </span>
                </div>
                <For each={rows()}>
                  {(row) => (
                    <div class="roster__row" role="row">
                      <span class="roster__cell roster__alias" role="cell">
                        {row.alias}
                      </span>
                      <span
                        classList={{
                          'roster__cell': true,
                          'roster__tg-link': true,
                          'roster__tg-link--on': telegramConnected(row),
                          'roster__tg-link--off': !telegramConnected(row),
                        }}
                        role="cell"
                      >
                        {telegramConnected(row) ? 'Linked' : 'Not linked'}
                      </span>
                      <span class="roster__cell roster__telegram" role="cell">
                        {telegramLabel(row)}
                      </span>
                      <span class="roster__cell roster__role" role="cell">
                        {row.role}
                      </span>
                    </div>
                  )}
                </For>
              </div>
            </>
          )}
        </Match>
      </Switch>
    </ConsoleCard>
  )
}
