import { For, createEffect, createSignal, onMount, type JSX } from 'solid-js'
import { Navigate, Route, Router, useLocation, useNavigate, type RouteSectionProps } from '@solidjs/router'
import { AuthDock } from './components/AuthDock'
import { NavRail, type NavRailId } from './components/NavRail'
import { PagePanels } from './components/PagePanels'
import { StageArt } from './components/StageArt'
import { authReady, bootAuthSession, homePath } from './lib/auth'
import { LAYER_ROUTE_PATHS, PageLayerContent } from './lib/pageRegistry'
import {
  NAV_PATHS,
  guardAdminPath,
  normalizePath,
  pathToNavId,
} from './lib/routes'

type PageLayer = {
  key: string
  path: string
  exiting: boolean
  staggered?: boolean
}

/**
 * Router owns the URL (null Routes). StageShell owns visible pages + slide layers.
 * Path → page lives in `pageRegistry` so Routes and layer content stay in sync.
 */
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
    <>
      {/* Outside .stage so overflow/isolation cannot stretch cover to document height */}
      <StageArt />
      <div class="stage">
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
                <PageLayerContent path={layer.path} ready={authReady()} />
              </PagePanels>
            )}
          </For>
        </div>

        <div hidden>{props.children}</div>
      </div>
    </>
  )
}

function HomeRedirect(): JSX.Element {
  return <Navigate href={homePath()} />
}

function App() {
  return (
    <Router root={StageShell} base={import.meta.env.BASE_URL.replace(/\/$/, '')}>
      <Route path="/" component={HomeRedirect} />
      {/* Paths from LAYER_ROUTE_PATHS — keep in sync via pageRegistry */}
      <Route path={LAYER_ROUTE_PATHS[0]} component={() => null} />
      <Route path={LAYER_ROUTE_PATHS[1]} component={() => null} />
      <Route path={LAYER_ROUTE_PATHS[2]} component={() => null} />
      <Route path={LAYER_ROUTE_PATHS[3]} component={() => null} />
      <Route path={LAYER_ROUTE_PATHS[4]} component={() => null} />
      <Route path={LAYER_ROUTE_PATHS[5]} component={() => null} />
      <Route path={LAYER_ROUTE_PATHS[6]} component={() => null} />
      <Route path={LAYER_ROUTE_PATHS[7]} component={() => null} />
      <Route path={LAYER_ROUTE_PATHS[8]} component={() => null} />
      <Route path="/tournaments" component={() => <Navigate href={NAV_PATHS.fantasy} />} />
      <Route path="*404" component={HomeRedirect} />
    </Router>
  )
}

export default App
