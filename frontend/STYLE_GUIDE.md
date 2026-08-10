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
| `--color-metal` | `#1a1a1a` | Metallic plate base |
| `--color-metal-hi` | `#2c2c2c` | Metal highlight edge |
| `--color-metal-lo` | `#0d0d0d` | Metal shadow edge |

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
| `--color-info-border` | `#444459` | Neutral info panels |
| `--color-info-bg` | `#4e4e5820` | Neutral info fill |

### Race palettes

Scoped to player/race UI (count chips, future participant rows). Icons live in `src/assets/races/`.

| Race | Background | Text | Link | Link hover |
|---|---|---|---|---|
| Protoss | `--color-race-protoss-bg` `#121008` | `#f0e6c8` | `#e8c547` | `#ffe070` |
| Terran | `--color-race-terran-bg` `#0a0f1e` | `#d6e4ff` | `#4a7cff` | `#7aa8ff` |
| Zerg | `--color-race-zerg-bg` `#140808` | `#f0d0d0` | `#e03030` | `#ff5a4a` |
| Random | `--color-race-random-bg` `#100c08` | `#e8dcc8` | `#c4893a` | `#e0a85a` |

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

### Metallic plate (grim)

Vertical brushed gradient + hard inner rim:

```css
background: linear-gradient(
  180deg,
  var(--color-metal-hi) 0%,
  var(--color-metal) 42%,
  var(--color-metal-lo) 100%
);
border: 1px solid var(--color-border);
```

### Dual-layer: metal rim + glass belly

Used by the main console (and nested telemetry):

1. **Rim (opaque)** — `.console__frame`: metallic plate + red border + hazard top rail + corner brackets.
2. **Belly (glass)** — `.console__inner` / `.telemetry`:

```css
background: rgba(0, 0, 0, 0.72);
backdrop-filter: blur(2px);
border: 1px solid var(--color-border); /* telemetry only; inner inherits frame */
```

Art and CRT read through the belly; chrome stays solid so the panel still feels Brood War industrial.

### Yellow hazard stripe rail

Diagonal caution tape for headers / plate edges:

```css
background: repeating-linear-gradient(
  -45deg,
  #000000 0 8px,
  var(--color-hazard) 8px 16px
);
```

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
| Display | Michroma (Eurostile stand-in) | Titles, buttons, labels — uppercase |
| UI | Source Sans 3 | Body / subtitle |
| Mono | Source Code Pro | URLs, telemetry, chips |

Title text uses `--color-text` with a light green glow; accent words use `--color-theme`.

---

## 5. Component mapping

| UI piece | Recipe |
|---|---|
| Stage | Art (`home.jpg`) + grid + CRT + scan + vignette |
| Console frame | Metallic rim + red border + hazard top rail (opaque) |
| Console inner / telemetry | Glass belly `rgba(0,0,0,~0.72)` + light blur |
| Corners | `--color-border-hot` L-brackets |
| Chips | Blacked panel, red border; live dot = theme green |
| Input | Blacked well, red border; focus = hot red + green wash |
| Valid input | Theme green border |
| Invalid input | Hot red border |
| Primary button | Flat theme green at rest; CRT scanlines + energy sweep on hover only |
| Ghost button | Transparent, red border; hover only shifts border/text — no gradient or CRT |
| Telemetry | Glass belly, green header, hazard stripe rule, red separators |
| Tabs | Spec only — see §3 Console tabs |
| Info / idle hint | Info slate border/bg |
| Error status | Hot red text |
| Success status | Theme green text |

---

## 6. Motion

Keep intentional and sparse:

1. **Console enter** — short fade/rise.
2. **CRT scan drift** — continuous, low opacity.
3. **Primary button sweep** — green energy on hover only.
4. **Busy pulse** — green rim while transmitting.

No bounce, no purple glow, no soft shadow stacks.

---

## 7. Checklist for new UI

- [ ] Uses tokens from this guide only  
- [ ] Borders lean red; accents lean green  
- [ ] Panels are black or metallic — not blue navy  
- [ ] Hazard stripes only on trim  
- [ ] CRT visible on full-bleed stages  
- [ ] Angular clip paths, not rounded cards  
