# ASL Uplink — Visual Style Guide

Brood War Remastered command-console aesthetic: **neon green + blood red** on black iron, with grim metallic hazard striping and CRT scan texture.

References: classic StarCraft / Brood War UI chrome, [StarCraft: Remastered marketing](https://starcraft.blizzard.com/en-us/) greens, Terran red frame lines.

---

## 1. Design principles

1. **Grim industrial** — scratched metal, black voids, no soft pastel UI.
2. **Green = live / select / success** — menu phosphor and active energy.
3. **Red = structure / danger** — panel frames, invalid state, hot borders.
4. **Yellow-black stripes = hazard / metal trim** — not a primary text color.
5. **CRT presence** — horizontal scanlines + drifting scan band on the stage.
6. **Angular clips** — hard corners, no pill shapes, no soft cards.
7. **No marketing fluff** — no large hero subtitles or explanatory blurb under the title. Labels, chips, fields, and status lines carry meaning; don’t pad the viewport with “what this page does” copy.
8. **Grey rules inside** — section breaks in the glass belly use thin grey `.rule` lines (`--color-rule`). Red stays on frames; hazard stays on trim.

---

## 2. Color tokens

Defined in `src/index.css` `@theme`. Prefer tokens; do not invent one-off hex in components.

### Surfaces (blacked panels)

| Token | Hex | Use |
|---|---|---|
| `--color-void` | `#000000` | Page / CRT glass |
| `--color-void-soft` | `#050505` | Secondary stage depth |
| `--color-panel` | `#0a0a0a` | Main console fill |
| `--color-panel-raised` | `#121212` | Nested blocks, chips |
| `--color-metal` | `#3a3a3a` | Metallic plate base |
| `--color-metal-hi` | `#5a5a5a` | Metal highlight mid |
| `--color-metal-lo` | `#242424` | Metal shadow edge |
| `--color-metal-edge` | `#f2f2f2` | Hard silver rim highlight (1px, no soft gradient) |

### Theme green (accent)

| Token | Hex | Use |
|---|---|---|
| `--color-theme` | `#70fe3a` | Primary accent, CTAs, links, live |
| `--color-theme-light` | `#f0ff90` | Hover / highlight wash |
| `--color-theme-dim` | `#1e7f0d` | Glyphs, quiet green chrome |
| `--color-theme-glow` | `rgba(112, 254, 58, 0.28)` | Soft bloom |

### Brood War red (structure + alert)

| Token | Hex | Use |
|---|---|---|
| `--color-border` | `#850100` | Default panel / input borders |
| `--color-border-hot` | `#d50103` | Focus, active frame, error emphasis |
| `--color-alert` | `#d50103` | Error text / invalid shell |

### Hazard / metallic yellow

| Token | Hex | Use |
|---|---|---|
| `--color-hazard` | `#c9a227` | Stripe yellow (with black) |
| `--color-hazard-dim` | `#7a6414` | Quiet metal gold text |

### Neutral text & info

| Token | Hex | Use |
|---|---|---|
| `--color-text` | `#e0e0e0` | Primary body / values |
| `--color-text-dim` | `#9a9a9a` | Secondary / idle |
| `--color-rule` | `#3a3a3a` | Interior section hairlines (`.rule`) |
| `--color-info-border` | `#444459` | Neutral info panels |
| `--color-info-bg` | `#4e4e5820` | Neutral info fill |

### Race palettes

Scoped to player/race UI (`Player` component, count chips). Icons live in `src/assets/races/`.

| Race | Background | Text | Link | Link hover |
|---|---|---|---|---|
| Protoss | `--color-race-protoss-bg` `#121008` | `#f0e6c8` | `#e8c547` | `#ffe070` |
| Terran | `--color-race-terran-bg` `#0a0f1e` | `#d6e4ff` | `#4a7cff` | `#7aa8ff` |
| Zerg | `--color-race-zerg-bg` `#140808` | `#f0d0d0` | `#e03030` | `#ff5a4a` |
| Random | `--color-race-random-bg` `#100c08` | `#e8dcc8` | `#c4893a` | `#e0a85a` |

**`Player`** — race icon + name; profile links use that race’s link/hover tokens (not theme green). Excluded players are struck + dimmed with an `excluded` tag.

**Protoss icon recolor:** paint the emblem to **`#e8c547`** (primary), with optional highlight **`#ffe070`** and shadow toward **`#7a6414`**. Do not use theme green — that stays global UI only.

### Forbidden

- Cyan / navy “Protoss HUD” accents (`#4ec4ff`, blue steel panels).
- Soft coral alerts (`#ff5a4e`) — use `--color-border-hot` instead.
- Pure white large surfaces; cream / purple marketing themes.
- Using race green for Protoss (collides with `--color-theme`).

---

## 3. Surface recipes

### Blacked panel

```css
background: var(--color-panel);
border: 1px solid var(--color-border);
box-shadow:
  inset 0 1px 0 rgba(255, 255, 255, 0.04),
  inset 0 -2px 8px rgba(0, 0, 0, 0.65);
```

### Metallic plate (grim) — CSS-only for now

Grey plate with fake grain (stacked micro-gradients). **Rim highlight is a hard 1px silver edge** (`border-top: 1px solid var(--color-metal-edge)`) with a hard black recess under it — not a soft white gradient fade (match `terran_metal.png`).

```css
border-top: 1px solid var(--color-metal-edge); /* #f2f2f2 */
box-shadow: inset 0 1px 0 #000; /* hard separation under the lip */
```

**Later / now:** `src/assets/textures/dirt.webp` (grayscale grit map):

| Surface | Treatment | Size / offset |
|---|---|---|
| Outer metal | `soft-light` grit | ~320px @ 12% 8% |
| Mid rails | soft-light | ~200px @ -40px 55% |
| Corner plates | soft-light | ~100–110px, unique positions |
| Top caps | soft-light | ~220px, start vs end offsets |
| Hazard stripes | grit image `multiply` @ ~70% (no solid brown wash) | ~500px @ 28% 10% |
| Outer corners (sparse) | brown mask multiply ~12% | ~140px |
| Silver lip / glass belly | **no dirt** | — |

### Dual-layer: metal rim + glass belly

Used by `ConsoleCard` (main card):

1. **Rim (opaque, multi-div)** — `.console__shell`:
   - Outer stepped grey metal (`.console__metal--outer`)
   - Top bar: metal caps + **hazard only in the middle** (`.console__hazard`) — default
   - **Right-hazard variant** (`hazard="right"` / `.console--hazard-right`): plain metal top + vertical `.console__hazard--rail` on the right edge (side panels)
   - Mid plate with **thin right/bottom** rails and thicker corner blocks
   - Dark well (`.console__well`) then **red inner border** (`.console__red`) with padding — no L-corner brackets
   - Optional **top chrome** (`.console__header`) for nav / bars above the glass
2. **Belly (glass)** — `.console__inner` / `.telemetry`:

```css
background: rgba(0, 0, 0, 0.52);
backdrop-filter: blur(2px);
/* few-pixel edge vignette via ::before inset shadow + soft radial */
```

```tsx
<ConsoleCard top={<nav>…</nav>}>
  {/* glass body */}
</ConsoleCard>

{/* Side draft panel — drops from top, hazard on the right */}
<ConsoleCard class="console--side motion-drop-in" hazard="right">
  {/* groups + cost */}
</ConsoleCard>
```

Reference silhouette: `assets/terran_metal.png`.

### Yellow hazard stripe rail

Base caution tape stays readable yellow/black:

```css
background: repeating-linear-gradient(-45deg, #050505 0 7px, #c9a227 7px 14px);
```

Dirt: `::before` with `dirt.webp` at low opacity + `mix-blend-mode: multiply` (dark grit only dirty the yellow; no solid brown fill / mask wash). Recess with inset shadows.

Use sparingly: top rail of the console, **right rail on side consoles**, section dividers — not full backgrounds.

### CRT stage + background art

Layer order (bottom → top):

1. `.stage__art` — full-bleed photograph, **`position: fixed; inset: 0`** (browser window), mounted **outside** `.stage` so document/body growth cannot restretch `cover`. Dimmed (`brightness` ~0.48, slight desat). **Per-route asset** — see §3a. `.stage` background is transparent so the fixed art shows through.
2. Art darken wash (gradient overlay).
3. `.stage__grid` — animated green grid (masked).
4. `.stage__crt` — horizontal scanlines.
5. `.stage__scan` — drifting green band.
6. `.stage__vignette` — edge crush.
7. `.console` — metal rim + glass belly.

Stage grows with content (`min-height: 100svh`). Use `overflow: clip` on `.stage` so backdrop transforms/scan can't inflate `<body>` scroll height, and so there is no nested scrollbar. Page scroll follows in-flow content only. Scan band runs the full stage (`top: -26% → 100%`); `.stage__scan { overflow: hidden }` clips the band at the edges.

Keep grid/CRT/scan when swapping art; only replace the art asset (and its fade handoff). Source catalog: [`assets/background/BACKGROUND.md`](./assets/background/BACKGROUND.md) (library under `frontend/assets/background/`; runtime copies live under `src/assets/background/` as needed).

### 3a. Per-page backgrounds

Every routed channel gets its **own** stage art. Do not reuse one global `home.jpg` for all pages once multi-page backgrounds are wired.

#### Rules

1. **One unique image per distinct page** — `/parser`, `/players`, `/fantasy-league`, and any future top-level channel each pick a different file from the catalog.
2. **Similar pages → similar palettes** — group by tone family from `BACKGROUND.md`, not by filename order. Nested or sibling views that feel like the same channel (e.g. player detail under Players) stay in that family’s palette even if the exact file differs.
3. **Main pages → best tone/content fit** — choose the catalog entry whose **atmosphere + content** match the channel’s job (console work, roster/people, epic league), not merely a pretty frame. Prefer the “Strong fits” rows in `BACKGROUND.md`.
4. **Fade through black** — on route change, art does **not** cross-dissolve image→image. Sequence: current art → fade to void black → new art fades in. Keep CRT/grid/scan/vignette mounted; only the art layer opacity (or a black scrim over art) animates. Duration should feel deliberate but shorter than card slide (`--motion-slide-duration`); reuse a shared token (e.g. `--motion-bg-fade-duration`) rather than one-offs.
5. **Legibility first** — catalog notes on bright beams / logo bands still apply; glass belly + darken wash stay. Do not pick a busier hero just because it is “more epic” if UI contrast dies.
6. **UI chrome unchanged** — green/red/metal tokens stay global. Background palette informs *which art* to pick, not a second HUD color system (no cyan Protoss panels, no purple marketing washes on chrome).

#### Palette families (from catalog)

| Family | Feel | Typical hex anchors | Use for |
|---|---|---|---|
| Terran industrial | Dust, steel, amber lamps, phosphor | charcoal, ochre, amber, CRT green | Parser, bunker/console, settings-like |
| Terran war / roster | Marines, rain, battlecruiser void | gunmetal, planet blue, fire orange | Players, human-facing lists |
| Cosmic desolate | Moon, desert plain, cold nebula | indigo, silver moon, dusty brown | Neutral default / uplink calm |
| Epic void war | Carriers, bombardment, fleet scale | void black, beam cyan, lava/gold | Fantasy League, competition |
| Protoss sacred | Gold hull, cyan psi, temple sky | bronze, cyan, blood-sky red | Future Protoss-tagged views |
| Zerg organic | Creep, hive, hydralisk, swarm dust | umber, maroon, magenta, sepia | Future Zerg-tagged views |

#### Canonical main-channel picks

Lock these unless the catalog gains a clearly better tone match — then update this table and `BACKGROUND.md` together.

| Route | Job | Family | Preferred art (catalog id / file) |
|---|---|---|---|
| `/parser` | Liquipedia uplink / console work | Terran industrial (+ cosmic OK) | `010` bunker CRT · calm alt `015` / `home.jpg` |
| `/me` | Operator profile / identity | Terran industrial (sibling of parser) | `002` industrial towers |
| `/players` | Roster / people | Terran war / roster | `terran-01` / `terran.png` · alt `009` rain marine · fleet alt `003` |
| `/fantasy-league` | Competition / league scale | Epic void war | `001` purification · alt `sc03` bombardment · `007` / `sc01` |

Sibling pages under a channel inherit that row’s **family**; pick a different file in-family so the fade still changes the picture.

### Console tabs

Original StarCraft / Remastered endgame style: mechanical chips, not modern underlines.

| State | Look |
|---|---|
| Idle | Dark translucent fill, dim text, thin `--color-border` |
| Active | `--color-theme` text, stronger border, bottom edge open to panel |
| Shape | Angular / notched top via clip-path |

Used in tournament telemetry (`Overview` / `Players` / `Results`): `.console__tabs` + `.tab` / `.tab--active`.

---

## 4. Typography

| Role | Family | Notes |
|---|---|---|
| Title | Orbitron (700) | Page hero title — SC menu geometry |
| Display | Michroma (Eurostile stand-in) | Labels — uppercase |
| Buttons | Orbitron (700) | Primary / ghost CTAs — uppercase |
| UI | Source Sans 3 | Body / field text |
| Mono | Source Code Pro | URLs, telemetry, chips |

Title text uses `--font-title` / `--color-text` with a light green glow; accent words use `--color-theme`.

---

## 5. Atmosphere (sparse HUD life)

Small living details sell the console — use them, but **never stack the same trick**.

### Budget

| Rule | Guidance |
|---|---|
| One beacon per region | At most **one** status light (green or red) in a meta row / header cluster |
| No twin dots | Never place two beacons side-by-side (green+red, or two greens) |
| Swap, don’t pile | Fault replaces live (`Channel open` → `Channel fault`) — don’t show both |
| Mix families | Pair a beacon with a *different* flourish (phosphor breath, hazard rule, CRT stage) — not another pulse-dot |
| Sparse motion | Prefer 1–2 atmospheric motions per viewport besides stage CRT |

### Flourish catalog

| Flourish | Class / recipe | Feel | Use |
|---|---|---|---|
| Live beacon | `.chip--live` | Soft green pulse (~2.4s) | Nominal / channel open / connected |
| Alert beacon | `.chip--alert` | Faster red pulse (~1s) | Fault / danger / uplink error |
| Phosphor breath | `.atm-phosphor` | Slow opacity breathe on text | One eyebrow or idle chrome line |
| Hazard micro-rule | `.brand__eyebrow::before` | Static gold tick | Section identity |
| CRT stage | `.stage__*` | Scan + grid + vignette | Full-bleed page backdrop only |
| Energy sweep | primary `.btn:hover` | Green crawl | CTA hover — not chips |
| Lamp flash | `installLampFlash` → `.lamp-flash` | Bright wash + pip on the pressed control only | Confirm on **any** enabled button press |

### Anti-patterns

- Multiple glowing dots in the same row or card header  
- Putting beacons on every chip “for consistency”  
- Pulsing body copy or values  
- Purple / neon stacks, bounce, or soft UI glow clouds  

---

## 6. Component mapping

| UI piece | Recipe |
|---|---|
| Stage | `StageArt` — per-route art (§3a) + grid + CRT + scan + vignette; art swaps via fade-to-black |
| Console frame | `ConsoleCard` — metal rim + hazard + red border; `top` slot; `hazard="right"` for side panels; optional `slide` for solo card motion; route pages use `PagePanels` |
| Console inner / telemetry | Glass belly ~52% black + few-px edge vignette + light blur |
| Page panels | `PagePanels` — one or more consoles slide in (right) / out (left) together on route change |
| Fantasy groups side panel | `FantasyGroupsPanel` inside `console--side` + `drop` motion + `hazard="right"`; on wide view shares the main grid cell and `margin-left` docks it to the **right** (main width/position unchanged; tall panel grows page height for scroll); on narrow viewports stacks **below** the main card so the team-editor caps line stays visible |
| Chips | Blacked panel, red border; live = green beacon; alert = red beacon (see §5) |
| Input | Blacked well, red border; focus = hot red + green wash |
| Checkbox | `.sc-check` — angular void plate, red rim; checked = theme green tick + glow |
| Valid input | Theme green border |
| Invalid input | Hot red border |
| Primary button | Flat theme green at rest; CRT scanlines + energy sweep on hover only |
| Ghost button | Transparent, red border; hover only shifts border/text — no gradient or CRT |
| Action dock | `ActionDock` — mini Ok/Cancel console shells; mount only when a confirm/dismiss action is required (not on the uplink form by default) |
| Auth dock | `AuthDock` — floating Login / alias plate (deep drop shadow above stage) |
| Nav rail | `NavRail` — left metal plates; green SVG icons; bottom strip ≤720px; floating drop-shadow; `visible` to hide; routes `/parser`, `/players`, `/fantasy-league` |
| Telemetry | Glass belly, green header, hazard stripe rule, grey interior block separators |
| Player | `Player` — race icon + race-colored profile link; excluded = strike + tag; hover dossier after ~360ms (shell first, then lookup + cached portrait) |
| Fantasy scores | `TeamScoreMeta` — PTS phosphor green + glow; COST quiet hazard-dim metal |
| Roster chip | `RosterPlayerChip` — pick-plate; race chrome; cost/pts; defeated muted; hover peer-lit with race color across teams |
| Champion | `ChampionMark` — pulsing phosphor star; hot green rim on winner chip/row |
| Interior separators | `.rule` — 1px `--color-rule` hairline inside glass; not red, not hazard |
| Tabs | `.console__tabs` / `.tab` — mechanical chips; used in tournament telemetry |
| Info / idle hint | Info slate border/bg |
| Error status | Hot red text |
| Success status | Theme green text |

**Nav rail mobile:** at `max-width: 720px` the rail becomes a fixed bottom strip (3 equal cells). Stage gains bottom padding so the console isn’t covered; ActionDock lifts above the strip when both are present.

**Routes:** `/` → `/parser`; `/players` and `/fantasy-league` are channel pages (fantasy is a placeholder). Nav selection navigates; exit starts immediately, enter follows after `--motion-slide-stagger` (220ms). `/tournaments` redirects to `/fantasy-league`.

---

## 7. Motion

Keep intentional and sparse.

### Metal slide (cards + nav with metal borders)

Applies to `ConsoleCard`, future metal-bordered navigation items, and any panel that travels on/off stage.

| Token / class | Value | Role |
|---|---|---|
| `--motion-slide-duration` | `600ms` | **Fixed** — same time for every slide, whether the travel is a short nudge or full off-screen |
| `--motion-slide-ease` | `cubic-bezier(0.05, 0.7, 0.1, 1)` | Ease-out: faster start, **slows at the end** (enter and exit) |
| `--motion-slide-stagger` | `220ms` | Delay before route enter so exit clears the stage without a long empty gap |
| `--motion-resize-duration` | `280ms` | Console shell height when content grows/shrinks — **shorter** than slide |
| `--motion-drop-duration` | `900ms` | **Exit only** — longer than slide (tall panels travel farther) |
| `--motion-drop-ease` | `cubic-bezier(0.55, 0.05, 0.9, 0.2)` | **Exit only** — ease-in: slow start, accelerates toward the end |
| `.motion-slide-in` | `translateX(100vw → 0)` | Enter from off-screen **right** |
| `.motion-slide-in--staggered` | + `animation-delay: stagger` | Route handoff enter only |
| `.motion-slide-out` | `translateX(0 → 100vw)` | Exit back off-screen **right** (reverse of enter) |
| `.motion-rise-in` | `translateY(100% → 0)` | Enter from off-screen **bottom** (action dock) |
| `.motion-rise-out` | `translateY(0 → 100%)` | Exit downward (reverse of rise) |
| `.motion-drop-in` | `translateY(calc(-100% - 100svh) → 0)` | Enter from off-screen **top** — uses slide duration + ease |
| `.motion-drop-out` | `translateY(0 → calc(-100% - 100svh))` | Exit upward — drop duration + ease; self height + viewport clears tall panels |

Rules:

1. **Constant duration** — do not scale time by distance. A short nav chip and a full card use the same `600ms`. Never invent per-element durations for the same motion family.
2. **Same curve in and out** — slide enter/exit both use `--motion-slide-ease`. Drop **enter** keeps slide timing; drop **exit** uses `--motion-drop-duration` / `--motion-drop-ease`.
3. **Exit reverses enter** — same axis and timing; play the enter path backwards (e.g. in from right → out to right). No fade for this family.
4. **Shared classes** — prefer `.motion-slide-in` / `.motion-slide-out` (or `PagePanels` `exiting`) instead of one-off keyframes.
5. **Resize ≠ slide** — panel height changes (telemetry appear/dismiss) use `--motion-resize-duration`, not slide duration.
6. **Route handoff stagger** — exit starts immediately; enter waits `--motion-slide-stagger` so panels don’t share the stage, without a long empty beat.
7. **Exit out of flow** — `.page-panels.motion-slide-out` is `position: absolute` so a tall outgoing page does not keep document height alive (avoids centering jump when the stage shrinks). Scroll resets to top on route change.

### Background art fade (route change)

| Token / behavior | Guidance |
|---|---|
| `--motion-bg-fade-duration` | `350ms` each half (out / in); shorter than slide |
| Out | Current `.stage__art` (or black scrim) → void `#000` |
| Hold | Brief black beat optional; avoid a long empty stage |
| In | New art opacity 0 → 1 from black |
| Forbidden | Direct crossfade between two photos; wiping/sliding the photograph with the console |

Coordinate with panel slide: art may start fading as soon as the route changes; consoles still follow §7 metal-slide rules.

### Other motion

1. **CRT scan drift** — continuous, low opacity.
2. **Primary button sweep** — green energy on hover only.
3. **Busy pulse** — green rim while transmitting.
4. **Live beacon** — slow green blink on `.chip--live`.
5. **Alert beacon** — faster red blink on `.chip--alert`.
6. **Phosphor breath** — slow opacity on `.atm-phosphor` (one line max nearby).
7. **Console resize** — `ConsoleCard` shell height eases when inner content size changes.
8. **Stage art fade** — through black on route background change (§3a).

No bounce, no purple glow, no soft shadow stacks.

---

## 8. Checklist for new UI

- [ ] Uses tokens from this guide only  
- [ ] Borders lean red; accents lean green  
- [ ] Checkboxes use `.sc-check` (no native OS chrome)  
- [ ] Floating chrome (nav / auth / login modal) casts a clear drop shadow above the stage  
- [ ] Panels are black or metallic — not blue navy  
- [ ] Hazard stripes only on trim  
- [ ] CRT visible on full-bleed stages  
- [ ] Page has its own stage art from `BACKGROUND.md`; family matches channel (§3a)  
- [ ] Background route change fades through black — no photo crossfade  
- [ ] Angular clip paths, not rounded cards  
- [ ] Atmosphere budget respected — no twin beacons / repeated flourishes  
- [ ] No large fluff subtitle under titles — chrome speaks through labels/status  
- [ ] Interior section breaks use grey `.rule` — reserve red for frames, hazard for trim  
- [ ] Metal slides use shared duration/ease; exit = reverse of enter (same axis)  
