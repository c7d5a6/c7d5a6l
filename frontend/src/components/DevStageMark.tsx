import { Show, type JSX } from 'solid-js'

/** Corner dev badge — dev server only (stripped from production builds). */
export function DevStageMark(): JSX.Element {
  return (
    <Show when={import.meta.env.DEV}>
      <div class="stage__dev-mark" aria-hidden="true">
        <span class="stage__dev-mark__dot" />
        <span class="stage__dev-mark__text">Dev build</span>
      </div>
    </Show>
  )
}
