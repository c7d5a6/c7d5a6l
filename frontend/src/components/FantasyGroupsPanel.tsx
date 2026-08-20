import { For, Match, Show, Switch, createMemo, type JSX } from 'solid-js'
import { GroupCard } from './GroupCard'
import { groupsByPhase } from '../lib/groupsByPhase'
import type { FantasyGroup } from '../types/fantasy'

export type FantasyGroupsPanelProps = {
  groups: FantasyGroup[]
  error?: unknown
}

/**
 * Tournament groups with fantasy cost only — shown beside team create/edit.
 * Parent mounts this after groups have loaded so the shell sizes to content.
 */
export function FantasyGroupsPanel(props: FantasyGroupsPanelProps): JSX.Element {
  const byPhase = createMemo(() => groupsByPhase(props.groups))

  return (
    <div class="fantasy-groups">
      <header class="fantasy-groups__head">
        <p class="brand__eyebrow">Draft · Groups</p>
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
          <For each={byPhase()}>
            {([phase, phaseGroups]) => (
              <div class="telemetry__block">
                <Show when={phase}>
                  <div class="telemetry__block-head">{phase}</div>
                </Show>
                <For each={phaseGroups}>
                  {(g) => (
                    <GroupCard
                      name={g.name}
                      players={g.players}
                      playerExtra={(p) => {
                        const fp = g.players.find(
                          (x) => (x.link ?? '') === (p.link ?? '') && (x.name ?? '') === (p.name ?? ''),
                        )
                        return fp ? (
                          <span class="fantasy-groups__cost" title="Cost">
                            {fp.cost}
                          </span>
                        ) : null
                      }}
                    />
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
