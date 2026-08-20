import type { JSX } from 'solid-js'
import { ActionPanel } from './ActionPanel'

export type ActionDockProps = {
  okLabel?: string
  cancelLabel?: string
  busy?: boolean
  exiting?: boolean
  onOk: () => void
  onCancel: () => void
  /** Fired when rise-out animation finishes (use to unmount after Cancel). */
  onExitEnd?: () => void
}

/**
 * Bottom-right Ok / Cancel dock.
 * Mini console shells: outer metal → hazard strip → mid rail → well → red → face.
 * Mount only when a confirm/dismiss action is required.
 */
export function ActionDock(props: ActionDockProps): JSX.Element {
  const okLabel = () => props.okLabel ?? 'Ok'
  const cancelLabel = () => props.cancelLabel ?? 'Cancel'
  const busy = () => Boolean(props.busy)

  function onAnimationEnd(e: AnimationEvent) {
    if (e.target !== e.currentTarget) return
    if (!props.exiting) return
    props.onExitEnd?.()
  }

  return (
    <div
      classList={{
        'action-dock': true,
        'motion-rise-in': !props.exiting,
        'motion-rise-out': Boolean(props.exiting),
      }}
      aria-label="Page actions"
      onAnimationEnd={onAnimationEnd}
    >
      <ActionPanel
        label={busy() ? 'Writing…' : okLabel()}
        kind="ok"
        disabled={busy() || Boolean(props.exiting)}
        onClick={() => props.onOk()}
      />
      <ActionPanel
        label={cancelLabel()}
        kind="cancel"
        disabled={busy() || Boolean(props.exiting)}
        onClick={() => props.onCancel()}
      />
    </div>
  )
}
