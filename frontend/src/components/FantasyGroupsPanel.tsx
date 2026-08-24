import { For, Match, Switch, createMemo, type JSX } from 'solid-js'
import { ChannelHead, RailSection } from './ChannelChrome'
import { GroupCard } from './GroupCard'
import { groupsByPhase } from '../lib/groupsByPhase'
import { isPlayoffsPhase } from '../lib/matchBoard'
import type { FantasyGroup } from '../types/fantasy'

export type FantasyGroupsPanelProps = {
  groups: FantasyGroup[]
  error?: unknown
  /** Fantasy defeated links (lowercase). */
  defeatedLinks?: Set<string>
}

/**
 * Tournament groups with fantasy cost only — shown beside team create/edit.
 * Parent mounts this after groups have loaded so the shell sizes to content.
 */
export function FantasyGroupsPanel(props: FantasyGroupsPanelProps): JSX.Element {
  const byPhase = createMemo(() => groupsByPhase(props.groups))

  return (
    <div class="fantasy-groups channel-stack">
      <ChannelHead tag="Draft · Groups" title="Groups" compact />

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
              <RailSection
                label="Phase"
                title={phase || '—'}
                hazard={isPlayoffsPhase(phase)}
              >
                <For each={phaseGroups}>
                  {(g) => (
                    <GroupCard
                      name={g.name}
                      playoff={isPlayoffsPhase(g.phase)}
                      players={g.players.map((p) => ({ ...p, isWinner: p.isGroupWinner }))}
                      defeatedLinks={props.defeatedLinks}
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
              </RailSection>
            )}
          </For>
        </Match>
      </Switch>
    </div>
  )
}
