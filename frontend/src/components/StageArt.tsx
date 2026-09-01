import { createEffect, createSignal, onCleanup, type JSX } from 'solid-js'
import { useLocation } from '@solidjs/router'
import { stageBackgroundForPath } from '../lib/stageBackgrounds'

/** Half of fade-through-black (out or in). Keep in sync with `--motion-bg-fade-duration`. */
const BG_FADE_MS = 350

/**
 * Full-bleed stage photograph. On route change: fade to void, swap art, fade in.
 * CRT/grid/vignette stay outside this node.
 */
export function StageArt(): JSX.Element {
  const location = useLocation()
  const initial = stageBackgroundForPath(location.pathname)
  const [src, setSrc] = createSignal(initial)
  const [opaque, setOpaque] = createSignal(true)

  /** Intended art for the current route (may differ from `src` mid-fade). */
  let target = initial
  let generation = 0
  let timer: ReturnType<typeof setTimeout> | undefined

  createEffect(() => {
    const next = stageBackgroundForPath(location.pathname)
    if (next === target) return
    target = next

    const gen = ++generation
    setOpaque(false)

    if (timer !== undefined) clearTimeout(timer)
    timer = setTimeout(() => {
      if (gen !== generation) return
      setSrc(next)
      requestAnimationFrame(() => {
        if (gen !== generation) return
        setOpaque(true)
      })
    }, BG_FADE_MS)
  })

  onCleanup(() => {
    generation += 1
    if (timer !== undefined) clearTimeout(timer)
  })

  return (
    <div
      class="stage__art"
      classList={{ 'stage__art--hidden': !opaque() }}
      style={{ 'background-image': `url("${src()}")` }}
      aria-hidden="true"
    />
  )
}
