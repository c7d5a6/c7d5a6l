import { Show, type JSX, type ParentProps } from 'solid-js'

export type ConsoleCardProps = ParentProps<{
  /** Optional top chrome inside the red frame (nav, tabs, toolbar). */
  top?: JSX.Element
  class?: string
  /** Exit slide (left) — same duration/ease as enter; opposite direction. */
  exiting?: boolean
}>

/**
 * Terran metal rim + hazard stripe + red inner border + glass belly.
 * Put page content in `children`; optional `top` for nav / bars above the glass.
 * Slide motion: `.motion-slide-in` / `.motion-slide-out` (fixed duration, ease-out).
 */
export function ConsoleCard(props: ConsoleCardProps) {
  const motion = () => (props.exiting ? 'motion-slide-out' : 'motion-slide-in')

  return (
    <div
      class={props.class ? `console ${motion()} ${props.class}` : `console ${motion()}`}
    >
      <div class="console__shell">
        <div class="console__metal console__metal--outer" aria-hidden="true" />
        <div class="console__top" aria-hidden="true">
          <div class="console__top-cap console__top-cap--start" />
          <div class="console__hazard" />
          <div class="console__top-cap console__top-cap--end" />
        </div>
        <div class="console__metal console__metal--mid" aria-hidden="true" />
        <div class="console__well">
          <div class="console__red">
            <Show when={props.top}>{(top) => <div class="console__header">{top()}</div>}</Show>
            <div class="console__inner">{props.children}</div>
          </div>
        </div>
      </div>
    </div>
  )
}
