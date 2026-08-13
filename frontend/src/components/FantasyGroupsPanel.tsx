import { For, Match, Show, Switch, createMemo, type JSX } from 'solid-js'
import { Player } from './Player'
import { authFetch } from '../lib/auth'
import type { FantasyGroup } from '../types/fantasy'

export async function fetchFantasyGroups(leagueId: number): Promise<FantasyGroup[]> {
  const res = await authFetch(`/api/fantasy-leagues/${leagueId}/groups`)
  if (!res.ok) throw new Error(`groups uplink failed (${res.status})`)
  const data = (await res.json()) as { groups: FantasyGroup[] }
  return data.groups ?? []
}

export type FantasyGroupsPanelProps = {
  groups: FantasyGroup[]
  error?: unknown
}

/**
 * Tournament groups with fantasy cost only — shown beside team create/edit.
 * Parent mounts this after groups have loaded so the shell sizes to content.
 */
export function FantasyGroupsPanel(props: FantasyGroupsPanelProps): JSX.Element {
  const groupsByPhase = createMemo(() => {
    const map = new Map<string, FantasyGroup[]>()
    for (const g of props.groups) {
      const list = map.get(g.phase) ?? []
      list.push(g)
      map.set(g.phase, list)
    }
    return [...map.entries()]
  })

  return (
    <div class="fantasy-groups">
      <header class="fantasy-groups__head">
        <p class="brand__eyebrow atm-phosphor">Draft · Groups</p>
        <h2 class="fantasy-groups__title">Groups</h2>
      </header>
      <hr class="rule" />

      <Switch>
        <Match when={props.error}>
          <p class="status status--error">
            {(props.error as Error)?.message ?? 'Failed to load groups'}
          </p>
        </Match>
        <Match when={props.groups.length === 0}>
          <p class="status status--idle">No groups parsed yet</p>
        </Match>
        <Match when={true}>
          <For each={groupsByPhase()}>
            {([phase, phaseGroups]) => (
              <div class="telemetry__block">
                <Show when={phase}>
                  <div class="telemetry__block-head">{phase}</div>
                </Show>
                <For each={phaseGroups}>
                  {(g) => (
                    <div class="fantasy-groups__card">
                      <div class="telemetry__group-name">{g.name}</div>
                      <ul class="telemetry__list telemetry__list--chips">
                        <For each={g.players}>
                          {(p) => (
                            <li class="fantasy-groups__player">
                              <Player
                                name={p.name}
                                link={p.link}
                                race={p.race}
                                excluded={p.excluded}
                              />
                              <span class="fantasy-groups__cost" title="Cost">
                                {p.cost}
                              </span>
                            </li>
                          )}
                        </For>
                      </ul>
                    </div>
                  )}
                </For>
              </div>
            )}
          </For>
        </Match>
      </Switch>
    </div>
  )
}
