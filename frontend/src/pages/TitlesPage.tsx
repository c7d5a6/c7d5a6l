import { ConsoleCard } from '../components/ConsoleCard'
import { UserTitles } from '../components/UserTitles'
import { For, Match, Show, Switch, createResource, createSignal, type JSX } from 'solid-js'
import { authFetch, isAdmin } from '../lib/auth'
import { fetchFantasyLeagues } from '../lib/api/fantasy'
import { readApiError } from '../lib/api/http'
import { fetchUsers } from '../lib/api/users'
import type { UserTitle, UserTitleKind } from '../types/user'
import { displayValue } from '../types/tournament'

async function fetchTitles(): Promise<UserTitle[]> {
  const res = await authFetch('/api/user-titles')
  if (!res.ok) throw new Error(await readApiError(res, `titles uplink failed (${res.status})`))
  const data = (await res.json()) as { titles: UserTitle[] }
  return data.titles ?? []
}

/** Awards list; admins can create and edit titles. */
export function TitlesPage(): JSX.Element {
  const [titles, { refetch }] = createResource(fetchTitles)
  const [users] = createResource(() => isAdmin(), async (admin) => {
    if (!admin) return []
    return fetchUsers()
  })
  const [leagues] = createResource(() => isAdmin(), async (admin) => {
    if (!admin) return []
    return fetchFantasyLeagues()
  })

  const [editingId, setEditingId] = createSignal<number | null>(null)
  const [userId, setUserId] = createSignal('')
  const [kind, setKind] = createSignal<UserTitleKind>('fantasy')
  const [name, setName] = createSignal('')
  const [leagueId, setLeagueId] = createSignal('')
  const [date, setDate] = createSignal('')
  const [file, setFile] = createSignal<File | null>(null)
  const [clearImage, setClearImage] = createSignal(false)
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)

  function resetForm() {
    setEditingId(null)
    setUserId('')
    setKind('fantasy')
    setName('')
    setLeagueId('')
    setDate('')
    setFile(null)
    setClearImage(false)
    setError(null)
  }

  function startEdit(t: UserTitle) {
    setEditingId(t.id)
    setUserId(String(t.userId))
    setKind(t.kind)
    setName(t.name)
    setLeagueId(t.fantasyLeagueId != null ? String(t.fantasyLeagueId) : '')
    setDate(t.date ?? '')
    setFile(null)
    setClearImage(false)
    setError(null)
  }

  async function onSave(e: Event) {
    e.preventDefault()
    const uid = Number(userId())
    const n = name().trim()
    if (!uid || !n) return
    setBusy(true)
    setError(null)
    try {
      const body = new FormData()
      body.set('userId', String(uid))
      body.set('kind', kind())
      body.set('name', n)
      body.set('date', date())
      if (kind() === 'fantasy' && leagueId()) body.set('fantasyLeagueId', leagueId())
      if (clearImage()) body.set('image', new File([], ''))
      else if (file()) body.set('image', file()!)
      const id = editingId()
      const res = await authFetch(id == null ? '/api/user-titles' : `/api/user-titles/${id}`, {
        method: id == null ? 'POST' : 'PATCH',
        body,
      })
      if (!res.ok) throw new Error(await readApiError(res, `Save failed (${res.status})`))
      resetForm()
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setBusy(false)
    }
  }

  async function onDelete(id: number) {
    if (!window.confirm('Delete this title?')) return
    setBusy(true)
    setError(null)
    try {
      const res = await authFetch(`/api/user-titles/${id}`, { method: 'DELETE' })
      if (!res.ok) throw new Error(`delete failed (${res.status})`)
      if (editingId() === id) resetForm()
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <ConsoleCard class="console--wide">
      <header class="brand">
        <p class="brand__eyebrow atm-phosphor">Awards · Command Protocol</p>
        <h1 class="brand__title">
          Titles <span>Channel</span>
        </h1>
      </header>
      <hr class="rule" />

      <Show when={isAdmin()}>
        <form class="titles-form" onSubmit={(e) => void onSave(e)}>
          <label class="field">
            <span class="field__label">Operator</span>
            <span class="field__shell">
              <select
                class="field__input"
                value={userId()}
                disabled={busy()}
                onChange={(e) => setUserId(e.currentTarget.value)}
              >
                <option value="">Select user…</option>
                <For each={users() ?? []}>{(u) => <option value={u.id}>{u.alias}</option>}</For>
              </select>
            </span>
          </label>
          <label class="field">
            <span class="field__label">Kind</span>
            <span class="field__shell">
              <select
                class="field__input"
                value={kind()}
                disabled={busy()}
                onChange={(e) => setKind(e.currentTarget.value as UserTitleKind)}
              >
                <option value="fantasy">Fantasy league</option>
                <option value="tournament">Tournament</option>
              </select>
            </span>
          </label>
          <label class="field">
            <span class="field__label">Display name</span>
            <span class="field__shell">
              <input
                class="field__input"
                type="text"
                maxlength={80}
                value={name()}
                disabled={busy()}
                onInput={(e) => setName(e.currentTarget.value)}
              />
            </span>
          </label>
          <label class="field">
            <span class="field__label">Date</span>
            <span class="field__shell">
              <input
                class="field__input"
                type="date"
                value={date()}
                disabled={busy()}
                onInput={(e) => setDate(e.currentTarget.value)}
              />
            </span>
          </label>
          <Show when={kind() === 'fantasy'}>
            <label class="field">
              <span class="field__label">Fantasy league (optional)</span>
              <span class="field__shell">
                <select
                  class="field__input"
                  value={leagueId()}
                  disabled={busy()}
                  onChange={(e) => setLeagueId(e.currentTarget.value)}
                >
                  <option value="">Unlinked</option>
                  <For each={leagues() ?? []}>
                    {(l) => (
                      <option value={l.id}>{displayValue(l.tournamentName) || `League ${l.id}`}</option>
                    )}
                  </For>
                </select>
              </span>
            </label>
          </Show>
          <label class="field">
            <span class="field__label">Image (optional)</span>
            <span class="field__shell">
              <input
                class="field__input"
                type="file"
                accept="image/png,image/jpeg,image/webp"
                disabled={busy() || clearImage()}
                onChange={(e) => setFile(e.currentTarget.files?.[0] ?? null)}
              />
            </span>
          </label>
          <Show when={editingId() != null}>
            <label class="titles-form__clear">
              <input
                type="checkbox"
                checked={clearImage()}
                disabled={busy()}
                onChange={(e) => {
                  setClearImage(e.currentTarget.checked)
                  if (e.currentTarget.checked) setFile(null)
                }}
              />
              Remove image
            </label>
          </Show>
          <div class="actions">
            <button type="submit" class="btn btn--primary" disabled={busy() || !userId() || !name().trim()}>
              {busy() ? 'Saving…' : editingId() != null ? 'Save title' : 'Create title'}
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
      </Show>

      <Switch>
        <Match when={titles.loading}>
          <p class="status status--idle">Locking titles uplink…</p>
        </Match>
        <Match when={titles.error}>
          <p class="status status--error">{(titles.error as Error)?.message ?? 'Titles uplink failed'}</p>
        </Match>
        <Match when={(titles() ?? []).length === 0}>
          <p class="status status--idle">No titles in database</p>
        </Match>
        <Match when={titles()}>
          {(rows) => (
            <div
              classList={{ roster: true, 'roster--titles': true, 'roster--titles-admin': isAdmin() }}
              role="table"
              aria-label="User titles"
            >
              <div class="roster__head" role="row">
                <span class="roster__cell roster__mark" role="columnheader">
                  Mark
                </span>
                <span class="roster__cell roster__name" role="columnheader">
                  Name
                </span>
                <span class="roster__cell roster__operator" role="columnheader">
                  Operator
                </span>
                <span class="roster__cell roster__date" role="columnheader">
                  Date
                </span>
                <Show when={isAdmin()}>
                  <span class="roster__cell roster__actions" role="columnheader">
                    <span class="points-board__sr-only">Actions</span>
                  </span>
                </Show>
              </div>
              <For each={rows()}>
                {(row) => (
                  <div class="roster__row" role="row">
                    <span class="roster__cell roster__mark" role="cell">
                      <UserTitles titles={[row]} />
                    </span>
                    <span class="roster__cell roster__name" role="cell">
                      {row.name}
                    </span>
                    <span class="roster__cell roster__operator" role="cell">
                      {row.userAlias}
                    </span>
                    <span class="roster__cell roster__date" role="cell">
                      {displayValue(row.date)}
                    </span>
                    <Show when={isAdmin()}>
                      <span class="roster__cell roster__actions" role="cell">
                        <div class="roster__elo-actions">
                          <button
                            type="button"
                            class="btn btn--ghost btn--compact"
                            disabled={busy()}
                            onClick={() => startEdit(row)}
                          >
                            Edit
                          </button>
                          <button
                            type="button"
                            class="btn btn--ghost btn--compact"
                            disabled={busy()}
                            onClick={() => void onDelete(row.id)}
                          >
                            Delete
                          </button>
                        </div>
                      </span>
                    </Show>
                  </div>
                )}
              </For>
            </div>
          )}
        </Match>
      </Switch>
    </ConsoleCard>
  )
}
