import type { JSX } from 'solid-js'

/** Pulsing phosphor beacon — tournament group standings winner. */
export function GroupWinnerMark(): JSX.Element {
  return (
    <span
      class="group-winner-mark"
      title="Group winner"
      aria-label="Group winner"
    />
  )
}
