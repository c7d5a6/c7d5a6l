import { For, Match, Show, Switch, createEffect, createSignal, onMount, type JSX } from 'solid-js'
import { Navigate, Route, Router, useLocation, useNavigate, type RouteSectionProps } from '@solidjs/router'
import { AuthDock } from './components/AuthDock'
import { NavRail, type NavRailId } from './components/NavRail'
import { PagePanels } from './components/PagePanels'
import { StageArt } from './components/StageArt'
import { authReady, bootAuthSession, homePath, isAdmin } from './lib/auth'
import {
  NAV_PATHS,
  fantasyManageLeagueId,
  guardAdminPath,
  isFantasyManagePath,
  normalizePath,
  pathToNavId,
} from './lib/routes'
import { FantasyLeaguePage } from './pages/FantasyLeaguePage'
import { FantasyManageDetailPage } from './pages/FantasyManageDetailPage'
import { FantasyManagePage } from './pages/FantasyManagePage'
import { MePage } from './pages/MePage'
import { ParserPage } from './pages/ParserPage'
import { PlayersPage } from './pages/PlayersPage'
import { TitlesPage } from './pages/TitlesPage'
import { UsersPage } from './pages/UsersPage'

type PageLayer = {
  key: string
  path: string
  exiting: boolean
  staggered?: boolean
}

function StageShell(props: RouteSectionProps): JSX.Element {
  const location = useLocation()
  const navigate = useNavigate()
  let keySeq = 1

  const [layers, setLayers] = createSignal<PageLayer[]>([
    { key: '0', path: normalizePath(location.pathname), exiting: false },
  ])

  onMount(() => {
    void bootAuthSession()
  })

  createEffect(() => {
    if (!authReady()) return
    const redirect = guardAdminPath(location.pathname)
    if (redirect) {
      navigate(redirect, { replace: true })
    }
  })

  createEffect(() => {
    const path = normalizePath(location.pathname)
    const cur = layers()
    const live = cur.find((l) => !l.exiting)
    if (live?.path === path) return
    if (cur.some((l) => !l.exiting && l.path === path)) return

    window.scrollTo(0, 0)

    setLayers((prev) => [
      ...prev.map((l) => (l.exiting ? l : { ...l, exiting: true })),
      { key: String(keySeq++), path, exiting: false, staggered: true },
    ])
  })

  function removeLayer(key: string) {
    setLayers((prev) => prev.filter((l) => l.key !== key))
  }

  function onNavSelect(id: NavRailId) {
    const href = NAV_PATHS[id]
    if (normalizePath(location.pathname) === href) return
    navigate(href)
  }

  return (
    <div class="stage">
      <StageArt />
      <div class="stage__grid" aria-hidden="true" />
      <div class="stage__crt" aria-hidden="true" />
      <div class="stage__scan" aria-hidden="true" />
      <div class="stage__vignette" aria-hidden="true" />

      <NavRail visible activeId={pathToNavId(location.pathname)} onSelect={onNavSelect} />

      <AuthDock />

      <div class="stage__pages">
        <For each={layers()}>
          {(layer) => (
            <PagePanels
              exiting={layer.exiting}
              staggered={layer.staggered}
              onExitEnd={() => removeLayer(layer.key)}
            >
              <Switch>
                <Match when={layer.path === NAV_PATHS.parser && isAdmin()}>
                  <ParserPage />
                </Match>
                <Match when={layer.path === NAV_PATHS.players}>
                  <PlayersPage />
                </Match>
                <Match when={layer.path === NAV_PATHS.fantasy}>
                  <FantasyLeaguePage />
                </Match>
                <Match when={isFantasyManagePath(layer.path) && isAdmin()}>
                  <Show
                    when={fantasyManageLeagueId(layer.path)}
                    fallback={<FantasyManagePage />}
                  >
                    {(id) => <FantasyManageDetailPage leagueId={id()} />}
                  </Show>
                </Match>
                <Match when={layer.path === NAV_PATHS.users && isAdmin()}>
                  <UsersPage />
                </Match>
                <Match when={layer.path === NAV_PATHS.titles && isAdmin()}>
                  <TitlesPage />
                </Match>
                <Match when={layer.path === '/me'}>
                  <MePage />
                </Match>
              </Switch>
            </PagePanels>
          )}
        </For>
      </div>

      <div hidden>{props.children}</div>
    </div>
  )
}

function HomeRedirect(): JSX.Element {
  return <Navigate href={homePath()} />
}

function App() {
  return (
    <Router root={StageShell} base={import.meta.env.BASE_URL.replace(/\/$/, '')}>
      <Route path="/" component={HomeRedirect} />
      <Route path="/parser" component={() => null} />
      <Route path="/players" component={() => null} />
      <Route path="/fantasy-league" component={() => null} />
      <Route path="/fantasy-manage" component={() => null} />
      <Route path="/fantasy-manage/:id" component={() => null} />
      <Route path="/users" component={() => null} />
      <Route path="/titles" component={() => null} />
      <Route path="/me" component={() => null} />
      <Route path="/tournaments" component={() => <Navigate href={NAV_PATHS.fantasy} />} />
      <Route path="*404" component={HomeRedirect} />
    </Router>
  )
}

export default App
