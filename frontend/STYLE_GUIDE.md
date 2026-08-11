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
   - Top bar: metal caps + **hazard only in the middle** (`.console__hazard`)
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
```

Reference silhouette: `assets/terran_metal.png`.

### Yellow hazard stripe rail

Base caution tape stays readable yellow/black:

```css
background: repeating-linear-gradient(-45deg, #050505 0 7px, #c9a227 7px 14px);
```

Dirt: `::before` with `dirt.webp` at low opacity + `mix-blend-mode: multiply` (dark grit only dirty the yellow; no solid brown fill / mask wash). Recess with inset shadows.

Use sparingly: top rail of the console, section dividers — not full backgrounds.

### CRT stage + background art

Layer order (bottom → top):

1. `.stage__art` — full-bleed image (`src/assets/background/home.jpg`), `cover`, dimmed (`brightness` ~0.48, slight desat).
2. Art darken wash (gradient overlay).
3. `.stage__grid` — animated green grid (masked).
4. `.stage__crt` — horizontal scanlines.
5. `.stage__scan` — drifting green band.
6. `.stage__vignette` — edge crush.
7. `.console` — metal rim + glass belly.

Stage grows with content (`min-height: 100svh`). Use `overflow: clip` on `.stage` so backdrop transforms/scan can't inflate `<body>` scroll height, and so there is no nested scrollbar. Page scroll follows in-flow content only. Scan band stays within the stage (`top: -26% → 74%`).

Keep grid/CRT/scan when swapping art; only replace the art asset.

### Console tabs (spec only — not implemented yet)

Original StarCraft / Remastered endgame style: mechanical chips on the frame, not modern underlines.

| State | Look |
|---|---|
| Idle | Dark translucent fill, dim text, thin `--color-border` |
| Active | `--color-theme` text, stronger border, merges into glass belly (`border-bottom` transparent / shared fill) |
| Shape | Angular / notched top; sit on the console top edge, overlapping or replacing part of the hazard rail |
| Optional | Idle tabs may use race- or section-tinted borders (red / steel); active label always theme green |

Markup sketch for a future pass:

```html
<nav class="console__tabs" role="tablist">
  <button type="button" class="tab tab--active" role="tab" aria-selected="true">Parse</button>
  <button type="button" class="tab" role="tab" aria-selected="false">History</button>
</nav>
```

Do not ship tab components until product needs them; this section is the visual contract only.

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

### Anti-patterns

- Multiple glowing dots in the same row or card header  
- Putting beacons on every chip “for consistency”  
- Pulsing body copy or values  
- Purple / neon stacks, bounce, or soft UI glow clouds  

---

## 6. Component mapping

| UI piece | Recipe |
|---|---|
| Stage | Art (`home.jpg`) + grid + CRT + scan + vignette |
| Console frame | `ConsoleCard` — metal rim + hazard + red border; `top` slot; slide via `.motion-slide-in` / `exiting` → `.motion-slide-out` |
| Console inner / telemetry | Glass belly ~52% black + few-px edge vignette + light blur |
| Chips | Blacked panel, red border; live = green beacon; alert = red beacon (see §5) |
| Input | Blacked well, red border; focus = hot red + green wash |
| Valid input | Theme green border |
| Invalid input | Hot red border |
| Primary button | Flat theme green at rest; CRT scanlines + energy sweep on hover only |
| Ghost button | Transparent, red border; hover only shifts border/text — no gradient or CRT |
| Telemetry | Glass belly, green header, hazard stripe rule, grey interior block separators |
| Player | `Player` — race icon + race-colored profile link; excluded = strike + tag |
| Interior separators | `.rule` — 1px `--color-rule` hairline inside glass; not red, not hazard |
| Tabs | Spec only — see §3 Console tabs |
| Info / idle hint | Info slate border/bg |
| Error status | Hot red text |
| Success status | Theme green text |

---

## 7. Motion

Keep intentional and sparse.

### Metal slide (cards + nav with metal borders)

Applies to `ConsoleCard`, future metal-bordered navigation items, and any panel that travels on/off stage.

| Token / class | Value | Role |
|---|---|---|
| `--motion-slide-duration` | `600ms` | **Fixed** — same time for every slide, whether the travel is a short nudge or full off-screen |
| `--motion-slide-ease` | `cubic-bezier(0.05, 0.7, 0.1, 1)` | Ease-out: faster start, **slows at the end** |
| `.motion-slide-in` | `translateX(100vw → 0)` | Enter from off-screen **right** |
| `.motion-slide-out` | `translateX(0 → -100vw)` | Exit to off-screen **left** (opposite of enter) |

Rules:

1. **Constant duration** — do not scale time by distance. A short nav chip and a full card use the same `600ms`. Never invent per-element durations for the same motion family.
2. **Same curve in and out** — enter and exit both use `--motion-slide-ease`.
3. **Exit is reverse direction** — if enter comes from the right, exit leaves to the left (mirrored axis). No fade for this family.
4. **Shared classes** — prefer `.motion-slide-in` / `.motion-slide-out` (or `ConsoleCard` `exiting`) instead of one-off keyframes.

### Other motion

1. **CRT scan drift** — continuous, low opacity.
2. **Primary button sweep** — green energy on hover only.
3. **Busy pulse** — green rim while transmitting.
4. **Live beacon** — slow green blink on `.chip--live`.
5. **Alert beacon** — faster red blink on `.chip--alert`.
6. **Phosphor breath** — slow opacity on `.atm-phosphor` (one line max nearby).

No bounce, no purple glow, no soft shadow stacks.

---

## 8. Checklist for new UI

- [ ] Uses tokens from this guide only  
- [ ] Borders lean red; accents lean green  
- [ ] Panels are black or metallic — not blue navy  
- [ ] Hazard stripes only on trim  
- [ ] CRT visible on full-bleed stages  
- [ ] Angular clip paths, not rounded cards  
- [ ] Atmosphere budget respected — no twin beacons / repeated flourishes  
- [ ] No large fluff subtitle under titles — chrome speaks through labels/status  
- [ ] Interior section breaks use grey `.rule` — reserve red for frames, hazard for trim  
- [ ] Metal slides use shared duration/ease; exit = opposite direction of enter  
