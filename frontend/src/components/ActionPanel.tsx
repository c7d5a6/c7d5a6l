import type { JSX } from 'solid-js'

export type ActionPanelProps = {
  label: string
  kind: 'ok' | 'cancel'
  disabled?: boolean
  class?: string
  title?: string
  onClick: () => void
}

/**
 * Mini Terran plate shell shared by ActionDock and AuthDock.
 * Outer metal → hazard → mid rail → well → red → face button.
 */
export function ActionPanel(props: ActionPanelProps): JSX.Element {
  const className = () =>
    ['action-panel', `action-panel--${props.kind}`, props.class].filter(Boolean).join(' ')

  return (
    <div class={className()}>
      <div class="action-panel__metal action-panel__metal--outer" aria-hidden="true" />
      <div class="action-panel__hazard" aria-hidden="true" />
      <div class="action-panel__metal action-panel__metal--mid" aria-hidden="true" />
      <div class="action-panel__well">
        <div class="action-panel__red">
          <button
            type="button"
            class={`action-panel__btn action-panel__btn--${props.kind}`}
            disabled={props.disabled}
            title={props.title}
            onClick={() => props.onClick()}
          >
            {props.label}
          </button>
        </div>
      </div>
    </div>
  )
}
