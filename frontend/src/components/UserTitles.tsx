import { For, Show, type JSX } from 'solid-js'
import { titleImageSrc, type UserTitle } from '../types/user'

export type UserTitlesProps = {
  titles: UserTitle[] | undefined
}

/** Compact award glyphs next to a user alias. */
export function UserTitles(props: UserTitlesProps): JSX.Element {
  const list = () => props.titles ?? []
  return (
    <Show when={list().length > 0}>
      <span class="user-titles">
        <For each={list()}>{(t) => <UserTitleMark title={t} />}</For>
      </span>
    </Show>
  )
}

function UserTitleMark(props: { title: UserTitle }): JSX.Element {
  const t = () => props.title
  return (
    <span class="user-title" title={t().name} aria-label={t().name}>
      <Show when={t().hasImage} fallback={<TitleGlyph kind={t().kind} />}>
        <span class="user-title__hex" aria-hidden="true">
          <img class="user-title__photo" src={titleImageSrc(t().id)} alt="" />
          <svg class="user-title__hex-stroke" viewBox="0 0 24 24">
            <path d="M12 1.6 L21.4 7 V17 L12 22.4 L2.6 17 V7 Z" />
          </svg>
        </span>
      </Show>
    </span>
  )
}

function TitleGlyph(props: { kind: UserTitle['kind'] }): JSX.Element {
  const common = {
    class: 'user-title__glyph',
    viewBox: '0 0 24 24',
    fill: 'currentColor',
    'aria-hidden': true as const,
  }
  if (props.kind === 'fantasy') {
    return (
      <svg {...common}>
        <path d="M12 2.2 L14.6 8.6 L21.6 9.1 L16.4 13.8 L18 20.7 L12 17.2 L6 20.7 L7.6 13.8 L2.4 9.1 L9.4 8.6 Z" />
      </svg>
    )
  }
  return (
    <svg {...common}>
      <path d="M6.5 4 H17.5 V7.2 C17.5 10.4 15.1 13 12 13 C8.9 13 6.5 10.4 6.5 7.2 Z" />
      <path d="M6.5 5.2 H3.8 C3.8 8.4 5.6 10.6 8.2 11.3" />
      <path d="M17.5 5.2 H20.2 C20.2 8.4 18.4 10.6 15.8 11.3" />
      <path d="M11.1 13.1 H12.9 V16.4 H15.2 V18.2 H8.8 V16.4 H11.1 Z" />
      <path d="M8 19.2 H16 V21 H8 Z" />
    </svg>
  )
}
