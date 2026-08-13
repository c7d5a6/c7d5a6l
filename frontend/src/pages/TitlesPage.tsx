import { ConsoleCard } from '../components/ConsoleCard'
import { UserTitles } from '../components/UserTitles'
import { For, Match, Show, Switch, createResource, createSignal, type JSX } from 'solid-js'
import { authFetch } from '../lib/auth'
import type { AuthUser, UserTitle, UserTitleKind } from '../types/user'
import type { FantasyLeague } from '../types/fantasy'
import { displayValue } from '../types/tournament'

async function fetchTitles(): Promise<UserTitle[]> {
  const res = await authFetch('/api/user-titles')
  if (!res.ok) throw new Error(`titles uplink failed (${res.status})`)
  const data = (await res.json()) as { titles: UserTitle[] }
  return data.titles ?? []
}

async function fetchUsers(): Promise<AuthUser[]> {
  const res = await authFetch('/api/users')
  if (!res.ok) throw new Error(`users uplink failed (${res.status})`)
  const data = (await res.json()) as { users: AuthUser[] }
  return data.users ?? []
}

async function fetchLeagues(): Promise<FantasyLeague[]> {
  const res = await authFetch('/api/fantasy-leagues')
  if (!res.ok) throw new Error(`leagues uplink failed (${res.status})`)
  const data = (await res.json()) as { leagues: FantasyLeague[] }
  return data.leagues ?? []
}

function formError(res: Response, body: unknown, fallback: string): string {
  const data = body as { error?: string }
  if (data?.error) return data.error
  return `${fallback} (${res.status})`
}

/** Admin awards — list and edit user titles. */
export function TitlesPage(): JSX.Element {
  const [titles, { refetch }] = createResource(fetchTitles)
  const [users] = createResource(fetchUsers)
  const [leagues] = createResource(fetchLeagues)

  const [editingId, setEditingId] = createSignal<number | null>(null)
  const [userId, setUserId] = createSignal('')
  const [kind, setKind] = createSignal<UserTitleKind>('fantasy')
  const [name, setName] = createSignal('')
  const [leagueId, setLeagueId] = createSignal('')
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
      if (kind() === 'fantasy' && leagueId()) body.set('fantasyLeagueId', leagueId())
      if (clearImage()) body.set('image', new File([], ''))
      else if (file()) body.set('image', file()!)
      const id = editingId()
      const res = await authFetch(id == null ? '/api/user-titles' : `/api/user-titles/${id}`, {
        method: id == null ? 'POST' : 'PATCH',
        body,
      })
      if (!res.ok) {
        let parsed: unknown = null
        try {
          parsed = await res.json()
        } catch {
          /* ignore */
        }
        throw new Error(formError(res, parsed, 'Save failed'))
      }
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
                  {(l) => <option value={l.id}>{displayValue(l.tournamentName) || `League ${l.id}`}</option>}
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
            <div class="roster roster--titles" role="table" aria-label="User titles">
              <div class="roster__head" role="row">
                <span class="roster__cell" role="columnheader">
                  Mark
                </span>
                <span class="roster__cell" role="columnheader">
                  Name
                </span>
                <span class="roster__cell" role="columnheader">
                  Operator
                </span>
                <span class="roster__cell" role="columnheader">
                  Kind
                </span>
                <span class="roster__cell" role="columnheader">
                  League
                </span>
                <span class="roster__cell roster__actions" role="columnheader">
                  <span class="points-board__sr-only">Actions</span>
                </span>
              </div>
              <For each={rows()}>
                {(row) => (
                  <div class="roster__row" role="row">
                    <span class="roster__cell" role="cell">
                      <UserTitles titles={[row]} />
                    </span>
                    <span class="roster__cell" role="cell">
                      {row.name}
                    </span>
                    <span class="roster__cell" role="cell">
                      {row.userAlias}
                    </span>
                    <span class="roster__cell" role="cell">
                      {row.kind}
                    </span>
                    <span class="roster__cell" role="cell">
                      {row.fantasyLeagueName ?? (row.fantasyLeagueId != null ? `#${row.fantasyLeagueId}` : '—')}
                    </span>
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
