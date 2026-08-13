import type { JSX } from 'solid-js'

/** Pulsing phosphor star — tournament champion badge. */
export function ChampionMark(): JSX.Element {
  return (
    <span class="champ-mark" title="Tournament champion" aria-label="Champion">
      <svg class="champ-mark__star" viewBox="0 0 24 24" aria-hidden="true">
        <path
          fill="currentColor"
          d="M12 2.2 L14.6 9.1 L22 9.4 L16.2 14.1 L18.2 21.5 L12 17.4 L5.8 21.5 L7.8 14.1 L2 9.4 L9.4 9.1 Z"
        />
      </svg>
    </span>
  )
}
