import { onCleanup, onMount, Show, type JSX, type ParentProps } from 'solid-js'

export type ConsoleCardProps = ParentProps<{
  /** Optional top chrome inside the red frame (nav, tabs, toolbar). */
  top?: JSX.Element
  class?: string
  /**
   * Metal slide on this card. Default false — route pages use `PagePanels` for
   * enter/exit so multiple cards can move as one stack.
   */
  slide?: boolean
  /** Exit slide (left) — only when `slide` is true. */
  exiting?: boolean
  onExitEnd?: () => void
}>

/**
 * Terran metal rim + hazard stripe + red inner border + glass belly.
 * Put page content in `children`; optional `top` for nav / bars above the glass.
 * Content size changes animate shell height (shorter than slide).
 */
export function ConsoleCard(props: ConsoleCardProps) {
  const motion = () => {
    if (!props.slide) return ''
    return props.exiting ? 'motion-slide-out' : 'motion-slide-in'
  }

  let shell!: HTMLDivElement
  let body!: HTMLDivElement

  onMount(() => {
    const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    let lastHeight = shell.getBoundingClientRect().height
    let animating = false
    let endTimer = 0

    const clearInline = () => {
      shell.style.height = ''
      shell.style.transition = ''
      shell.style.overflow = ''
    }

    const finish = () => {
      clearInline()
      lastHeight = shell.getBoundingClientRect().height
      animating = false
    }

    const animateTo = (from: number, to: number) => {
      if (reduceMotion || Math.abs(from - to) < 2) {
        lastHeight = to
        return
      }

      animating = true
      window.clearTimeout(endTimer)
      shell.style.overflow = 'hidden'
      shell.style.transition = 'none'
      shell.style.height = `${from}px`
      void shell.offsetHeight
      shell.style.transition = `height var(--motion-resize-duration) var(--motion-slide-ease)`
      shell.style.height = `${to}px`

      const onEnd = (e: TransitionEvent) => {
        if (e.target !== shell || e.propertyName !== 'height') return
        shell.removeEventListener('transitionend', onEnd)
        window.clearTimeout(endTimer)
        finish()
      }
      shell.addEventListener('transitionend', onEnd)
      endTimer = window.setTimeout(() => {
        shell.removeEventListener('transitionend', onEnd)
        finish()
      }, 400)
    }

    const ro = new ResizeObserver(() => {
      if (animating) return
      clearInline()
      const natural = shell.getBoundingClientRect().height
      const from = lastHeight
      if (Math.abs(from - natural) < 2) {
        lastHeight = natural
        return
      }
      animateTo(from, natural)
    })

    ro.observe(body)
    onCleanup(() => {
      ro.disconnect()
      window.clearTimeout(endTimer)
    })
  })

  function onAnimationEnd(e: AnimationEvent) {
    if (e.target !== e.currentTarget) return
    if (!props.slide || !props.exiting) return
    props.onExitEnd?.()
  }

  return (
    <div
      class={props.class ? `console ${motion()} ${props.class}` : `console ${motion()}`}
      onAnimationEnd={onAnimationEnd}
    >
      <div class="console__shell" ref={shell}>
        <div class="console__metal console__metal--outer" aria-hidden="true" />
        <div class="console__top" aria-hidden="true">
          <div class="console__top-cap console__top-cap--start" />
          <div class="console__hazard" />
          <div class="console__top-cap console__top-cap--end" />
        </div>
        <div class="console__metal console__metal--mid" aria-hidden="true" />
        <div class="console__well">
          <div class="console__red" ref={body}>
            <Show when={props.top}>{(top) => <div class="console__header">{top()}</div>}</Show>
            <div class="console__inner">{props.children}</div>
          </div>
        </div>
      </div>
    </div>
  )
}
