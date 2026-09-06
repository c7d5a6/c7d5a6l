import { For, Show, createEffect, createSignal, onCleanup, type JSX } from 'solid-js'
import { fetchMergeCandidates, mergePlayers, type MergeCandidate } from '../lib/api/players'
import { invalidatePlayerInfo } from '../lib/playerHoverCache'
import type { PlayerRaceEntry } from '../types/tournament'

export type PlayerMergeModalProps = {
  main: PlayerRaceEntry
  onClose: () => void
  onMerged: () => void | Promise<void>
}

function displayName(row: Pick<PlayerRaceEntry, 'name' | 'link'>): string {
  return row.name?.trim() || row.link
}

function candidateLabel(c: MergeCandidate): string {
  return c.name?.trim() || c.link
}

/** Admin modal to merge another player account into the main roster row. */
export function PlayerMergeModal(props: PlayerMergeModalProps): JSX.Element {
  const [query, setQuery] = createSignal('')
  const [candidates, setCandidates] = createSignal<MergeCandidate[]>([])
  const [loading, setLoading] = createSignal(true)
  const [busy, setBusy] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)
  const [selectedId, setSelectedId] = createSignal<number | null>(null)

  let debounce: ReturnType<typeof setTimeout> | undefined

  async function loadCandidates(q: string) {
    setLoading(true)
    setError(null)
    try {
      const rows = await fetchMergeCandidates(props.main.playerId, q)
      setCandidates(rows)
      if (rows.length > 0 && !rows.some((r) => r.playerId === selectedId())) {
        setSelectedId(rows[0].playerId)
      }
      if (rows.length === 0) {
        setSelectedId(null)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load candidates')
      setCandidates([])
      setSelectedId(null)
    } finally {
      setLoading(false)
    }
  }

  createEffect(() => {
    const q = query()
    if (debounce) clearTimeout(debounce)
    debounce = setTimeout(() => {
      void loadCandidates(q)
    }, 200)
  })
  onCleanup(() => {
    if (debounce) clearTimeout(debounce)
  })

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && !busy()) props.onClose()
  }
  window.addEventListener('keydown', onKey)
  onCleanup(() => window.removeEventListener('keydown', onKey))

  async function onConfirm() {
    const aliasId = selectedId()
    if (!aliasId) return
    const alias = candidates().find((c) => c.playerId === aliasId)
    const aliasLabel = alias ? candidateLabel(alias) : 'selected player'
    const ok = window.confirm(
      `Merge "${aliasLabel}" into "${displayName(props.main)}"?\n\n` +
        'The main account keeps its rating and tournament ids. ' +
        'The other account becomes aliases and is deleted.',
    )
    if (!ok) return

    setBusy(true)
    setError(null)
    try {
      await mergePlayers(props.main.playerId, aliasId)
      invalidatePlayerInfo(props.main.link)
      if (alias?.link) invalidatePlayerInfo(alias.link)
      await props.onMerged()
      props.onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Merge failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      class="merge-modal"
      role="dialog"
      aria-modal="true"
      aria-labelledby="merge-modal-title"
      onClick={(e) => {
        if (e.target === e.currentTarget && !busy()) props.onClose()
      }}
    >
      <div class="merge-modal__panel motion-drop-in">
        <header class="merge-modal__head">
          <h2 id="merge-modal-title" class="merge-modal__title">Merge player</h2>
          <button
            type="button"
            class="btn btn--ghost btn--compact"
            disabled={busy()}
            onClick={() => props.onClose()}
          >
            Close
          </button>
        </header>

        <p class="merge-modal__hint">
          Keep <strong>{displayName(props.main)}</strong> as the main account. Pick another player
          to absorb as aliases.
        </p>

        <label class="field">
          <span class="field__label">Search players</span>
          <input
            class="field__input"
            type="search"
            placeholder="Name, alias, or link fragment"
            value={query()}
            disabled={busy()}
            onInput={(e) => setQuery(e.currentTarget.value)}
          />
        </label>

        <Show when={error()}>
          <p class="status status--error">{error()}</p>
        </Show>

        <div class="merge-modal__list" role="listbox" aria-label="Merge candidates">
          <Show when={loading()}>
            <p class="status status--idle">Scanning roster…</p>
          </Show>
          <Show when={!loading() && candidates().length === 0}>
            <p class="status status--idle">No matching players</p>
          </Show>
          <For each={candidates()}>
            {(c) => (
              <button
                type="button"
                classList={{
                  'merge-modal__row': true,
                  'merge-modal__row--selected': selectedId() === c.playerId,
                }}
                disabled={busy()}
                onClick={() => setSelectedId(c.playerId)}
              >
                <span class="merge-modal__row-main">
                  <span class="merge-modal__row-name">{candidateLabel(c)}</span>
                  <span class="merge-modal__row-meta">{c.matchReason}</span>
                </span>
                <Show when={c.aliases.length > 0}>
                  <span class="merge-modal__row-aliases">{c.aliases.join(' · ')}</span>
                </Show>
              </button>
            )}
          </For>
        </div>

        <div class="merge-modal__actions">
          <button
            type="button"
            class="btn btn--primary"
            disabled={busy() || selectedId() == null}
            onClick={() => void onConfirm()}
          >
            {busy() ? 'Merging…' : 'Merge into main'}
          </button>
        </div>
      </div>
    </div>
  )
}
