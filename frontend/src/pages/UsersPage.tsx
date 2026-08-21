import { ConsoleCard } from '../components/ConsoleCard'
import { ChannelHead, NestedPlate } from '../components/ChannelChrome'
import { For, Match, Show, Switch, createResource, createSignal, type JSX } from 'solid-js'
import { createUser, fetchUsers, updateUser, type UserWritePayload } from '../lib/api/users'
import type { AuthUser, UserRole } from '../types/user'

function telegramConnected(u: AuthUser): boolean {
  return u.telegramId != null
}

function telegramLabel(u: AuthUser): string {
  if (!telegramConnected(u)) return '—'
  if (u.telegramUsername) return `@${u.telegramUsername}`
  return `id ${u.telegramId}`
}

function emptyOrNull(s: string): string | null {
  const t = s.trim()
  return t ? t : null
}

function parseTelegramId(raw: string): number | null {
  const t = raw.trim()
  if (!t) return null
  const n = Number(t)
  if (!Number.isInteger(n) || n <= 0) return null
  return n
}

/** Admin roster — create/edit accounts (alias, name, telegram, role). */
export function UsersPage(): JSX.Element {
  const [users, { refetch }] = createResource(fetchUsers)
  const [editingId, setEditingId] = createSignal<number | null>(null)
  const [alias, setAlias] = createSignal('')
  const [firstName, setFirstName] = createSignal('')
  const [lastName, setLastName] = createSignal('')
  const [role, setRole] = createSignal<UserRole>('USER')
  const [telegramUsername, setTelegramUsername] = createSignal('')
  const [telegramId, setTelegramId] = createSignal('')
  const [photoUrl, setPhotoUrl] = createSignal('')
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  function resetForm() {
    setEditingId(null)
    setAlias('')
    setFirstName('')
    setLastName('')
    setRole('USER')
    setTelegramUsername('')
    setTelegramId('')
    setPhotoUrl('')
    setError(null)
  }

  function startEdit(u: AuthUser) {
    setEditingId(u.id)
    setAlias(u.alias)
    setFirstName(u.firstName)
    setLastName(u.lastName ?? '')
    setRole(u.role)
    setTelegramUsername(u.telegramUsername ?? '')
    setTelegramId(u.telegramId != null ? String(u.telegramId) : '')
    setPhotoUrl(u.photoUrl ?? '')
    setError(null)
  }

  function buildPayload(): UserWritePayload | null {
    const a = alias().trim()
    if (!a) return null
    const tgRaw = telegramId().trim()
    let tgId: number | null = null
    if (tgRaw) {
      tgId = parseTelegramId(tgRaw)
      if (tgId == null) {
        setError('Telegram id must be a positive integer')
        return null
      }
    }
    const fn = firstName().trim() || (editingId() == null ? a : '')
    if (!fn) {
      setError('First name is required')
      return null
    }
    return {
      alias: a,
      firstName: fn,
      lastName: emptyOrNull(lastName()),
      photoUrl: emptyOrNull(photoUrl()),
      telegramUsername: emptyOrNull(telegramUsername()),
      telegramId: tgId,
      role: role(),
    }
  }

  async function onSave(e: Event) {
    e.preventDefault()
    const payload = buildPayload()
    if (!payload) return
    setBusy(true)
    setError(null)
    try {
      const id = editingId()
      if (id == null) await createUser(payload)
      else await updateUser(id, payload)
      resetForm()
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <ConsoleCard class="console--wide">
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Operators · Command Protocol</p>
        <h1 class="brand__title">
          Users <span>Channel</span>
        </h1>
      </header>
      <hr class="rule" />

      <div class="channel-stack">
        <ChannelHead tag="Operators" title={editingId() != null ? 'Edit user' : 'Provision'} />
        <NestedPlate>
          <form class="users-form" onSubmit={(e) => void onSave(e)}>
            <label class="field">
              <span class="field__label">Alias</span>
              <span class="field__shell">
                <input
                  class="field__input"
                  type="text"
                  value={alias()}
                  maxlength={64}
                  placeholder="Operator callsign"
                  autocomplete="off"
                  disabled={busy()}
                  onInput={(e) => {
                    setAlias(e.currentTarget.value)
                    setError(null)
                  }}
                />
              </span>
            </label>
            <label class="field">
              <span class="field__label">First name</span>
              <span class="field__shell">
                <input
                  class="field__input"
                  type="text"
                  value={firstName()}
                  placeholder={editingId() == null ? 'Defaults to alias' : undefined}
                  autocomplete="off"
                  disabled={busy()}
                  onInput={(e) => {
                    setFirstName(e.currentTarget.value)
                    setError(null)
                  }}
                />
              </span>
            </label>
            <label class="field">
              <span class="field__label">Last name</span>
              <span class="field__shell">
                <input
                  class="field__input"
                  type="text"
                  value={lastName()}
                  autocomplete="off"
                  disabled={busy()}
                  onInput={(e) => {
                    setLastName(e.currentTarget.value)
                    setError(null)
                  }}
                />
              </span>
            </label>
            <label class="field">
              <span class="field__label">Role</span>
              <span class="field__shell">
                <select
                  class="field__input"
                  value={role()}
                  disabled={busy()}
                  onChange={(e) => {
                    setRole(e.currentTarget.value as UserRole)
                    setError(null)
                  }}
                >
                  <option value="USER">USER</option>
                  <option value="ADMIN">ADMIN</option>
                </select>
              </span>
            </label>
            <label class="field">
              <span class="field__label">Telegram username</span>
              <span class="field__shell">
                <input
                  class="field__input"
                  type="text"
                  value={telegramUsername()}
                  placeholder="without @"
                  autocomplete="off"
                  disabled={busy()}
                  onInput={(e) => {
                    setTelegramUsername(e.currentTarget.value)
                    setError(null)
                  }}
                />
              </span>
            </label>
            <label class="field">
              <span class="field__label">Telegram id</span>
              <span class="field__shell">
                <input
                  class="field__input"
                  type="text"
                  inputMode="numeric"
                  value={telegramId()}
                  placeholder="Optional numeric id"
                  autocomplete="off"
                  disabled={busy()}
                  onInput={(e) => {
                    setTelegramId(e.currentTarget.value)
                    setError(null)
                  }}
                />
              </span>
            </label>
            <label class="field users-form__photo">
              <span class="field__label">Photo URL</span>
              <span class="field__shell">
                <input
                  class="field__input"
                  type="url"
                  value={photoUrl()}
                  placeholder="https://…"
                  autocomplete="off"
                  disabled={busy()}
                  onInput={(e) => {
                    setPhotoUrl(e.currentTarget.value)
                    setError(null)
                  }}
                />
              </span>
            </label>
            <div class="actions">
              <button type="submit" class="btn btn--primary" disabled={busy() || !alias().trim()}>
                {busy() ? 'Saving…' : editingId() != null ? 'Save user' : 'Create user'}
              </button>
              <Show when={editingId() != null}>
                <button type="button" class="btn btn--ghost" disabled={busy()} onClick={resetForm}>
                  Cancel
                </button>
              </Show>
            </div>
          </form>
          <Show when={error()}>
            <p class="status status--error">{error()}</p>
          </Show>
        </NestedPlate>

        <ChannelHead tag="Roster" title="Users" />
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
                <div class="roster roster--users roster--users-admin" role="table" aria-label="Users">
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
                    <span class="roster__cell roster__actions" role="columnheader">
                      <span class="points-board__sr-only">Actions</span>
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
                        <span class="roster__cell roster__actions" role="cell">
                          <button
                            type="button"
                            class="btn btn--ghost btn--compact"
                            disabled={busy()}
                            onClick={() => startEdit(row)}
                          >
                            Edit
                          </button>
                        </span>
                      </div>
                    )}
                  </For>
                </div>
              </>
            )}
          </Match>
        </Switch>
      </div>
    </ConsoleCard>
  )
}
