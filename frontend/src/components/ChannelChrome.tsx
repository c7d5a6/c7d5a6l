import type { JSX, ParentProps } from 'solid-js'
import { Show } from 'solid-js'

export type ChannelHeadProps = {
  tag?: string
  title: string
  /** Hazard gold accent instead of phosphor green. */
  hazard?: boolean
  /** Compact for side panels / nested use. */
  compact?: boolean
  actions?: JSX.Element
}

/** Green-accent channel header (Results / admin / table sections). */
export function ChannelHead(props: ChannelHeadProps): JSX.Element {
  return (
    <header
      classList={{
        'channel-head': true,
        'channel-head--hazard': Boolean(props.hazard),
        'channel-head--compact': Boolean(props.compact),
      }}
    >
      <div class="channel-head__copy">
        <Show when={props.tag}>
          <span class="channel-head__tag">{props.tag}</span>
        </Show>
        <h2 class="channel-head__title">{props.title}</h2>
      </div>
      <Show when={props.actions}>
        <div class="channel-head__actions">{props.actions}</div>
      </Show>
    </header>
  )
}

export type RailSectionProps = ParentProps<{
  label?: string
  title: string
  hazard?: boolean
}>

/** Labeled hazard/phosphor rail wrapping a phase or subsection. */
export function RailSection(props: RailSectionProps): JSX.Element {
  return (
    <div
      classList={{
        'rail-section': true,
        'rail-section--hazard': Boolean(props.hazard),
      }}
    >
      <header class="rail-section__head">
        <span class="rail-section__rail" aria-hidden="true" />
        <div class="rail-section__copy">
          <Show when={props.label}>
            <span class="rail-section__label">{props.label}</span>
          </Show>
          <h3 class="rail-section__title">{props.title}</h3>
        </div>
      </header>
      <div class="rail-section__body">{props.children}</div>
    </div>
  )
}

export type NestedPlateProps = ParentProps<{
  class?: string
  hazard?: boolean
  live?: boolean
}>

/** Angular metal plate for nested lists / profile blocks. */
export function NestedPlate(props: NestedPlateProps): JSX.Element {
  return (
    <div
      classList={{
        'nested-plate': true,
        'nested-plate--hazard': Boolean(props.hazard),
        'nested-plate--live': Boolean(props.live),
      }}
      class={props.class}
    >
      {props.children}
    </div>
  )
}
