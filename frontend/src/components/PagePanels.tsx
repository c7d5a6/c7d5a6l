import type { JSX, ParentProps } from 'solid-js'

export type PagePanelsProps = ParentProps<{
  /** Exit slide — reverse of enter (back to the right). */
  exiting?: boolean
  /** Delay enter so it doesn’t share the stage with the outgoing page. */
  staggered?: boolean
  onExitEnd?: () => void
}>

/**
 * Route page stack — one or more ConsoleCards move in/out together.
 * Cards inside should use `slide={false}` so only this wrapper animates.
 */
export function PagePanels(props: PagePanelsProps): JSX.Element {
  function onAnimationEnd(e: AnimationEvent) {
    if (e.target !== e.currentTarget) return
    if (!props.exiting) return
    props.onExitEnd?.()
  }

  return (
    <div
      classList={{
        'page-panels': true,
        'motion-slide-in': !props.exiting,
        'motion-slide-in--staggered': !props.exiting && Boolean(props.staggered),
        'motion-slide-out': Boolean(props.exiting),
      }}
      onAnimationEnd={onAnimationEnd}
    >
      {props.children}
    </div>
  )
}
