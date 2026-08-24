import { Show, type JSX, type ParentProps } from 'solid-js'

export type ConsoleCardProps = ParentProps<{
  /** Optional top chrome inside the red frame (nav, tabs, toolbar). */
  top?: JSX.Element
  class?: string
  /**
   * Hazard stripe placement. Default `top` (main consoles).
   * `right` — vertical rail on the right edge (side panels).
   */
  hazard?: 'top' | 'right'
  /**
   * Metal slide on this card. Default false — route pages use `PagePanels` for
   * enter/exit so multiple cards can move as one stack.
   */
  slide?: boolean
  /**
   * Drop from top (enter) / rise back up (exit). Used for side panels.
   * Mutually exclusive with `slide` in practice — `drop` wins if both are set.
   */
  drop?: boolean
  /** Exit motion — only when `slide` or `drop` is true. */
  exiting?: boolean
  onExitEnd?: () => void
}>

/**
 * Terran metal rim + hazard stripe + red inner border + HUD belly.
 * Put page content in `children`; optional `top` for nav / bars above the glass.
 * Shell height follows content — no animated `height` (clips clip-path plates
 * and flickers on tab swaps, especially on mobile).
 */
export function ConsoleCard(props: ConsoleCardProps) {
  const motion = () => {
    if (props.drop) return props.exiting ? 'motion-drop-out' : 'motion-drop-in'
    if (!props.slide) return ''
    return props.exiting ? 'motion-slide-out' : 'motion-slide-in'
  }

  const hazardRight = () => (props.hazard ?? 'top') === 'right'

  function onAnimationEnd(e: AnimationEvent) {
    if (e.target !== e.currentTarget) return
    if (!props.exiting) return
    // Switching in→out cancels the enter animation and can fire animationend;
    // only unmount after the actual exit keyframes finish.
    if (props.drop) {
      if (e.animationName !== 'motion-drop-out') return
    } else if (props.slide) {
      if (e.animationName !== 'motion-slide-out') return
    } else {
      return
    }
    props.onExitEnd?.()
  }

  const className = () => {
    const parts = ['console', motion()]
    if (hazardRight()) parts.push('console--hazard-right')
    if (props.class) parts.push(props.class)
    return parts.filter(Boolean).join(' ')
  }

  return (
    <div class={className()} onAnimationEnd={onAnimationEnd}>
      <div class="console__shell">
        <div class="console__metal console__metal--outer" aria-hidden="true" />
        <Show
          when={hazardRight()}
          fallback={
            <div class="console__top" aria-hidden="true">
              <div class="console__top-cap console__top-cap--start" />
              <div class="console__hazard" />
              <div class="console__top-cap console__top-cap--end" />
            </div>
          }
        >
          <div class="console__top console__top--plain" aria-hidden="true">
            <div class="console__top-cap console__top-cap--start" />
            <div class="console__top-fill" />
            <div class="console__top-cap console__top-cap--end" />
          </div>
          <div class="console__hazard console__hazard--rail" aria-hidden="true" />
        </Show>
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
