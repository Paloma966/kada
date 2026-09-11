# KADA homepage 3D starfield restyle design

date: 2026-08-10
status: confirmed with the user

## goals

restyle the homepage `/` into a **single-screen, no-scroll, full-viewport immersive 3D starfield page**:

- the background subject is the **Ophiuchus constellation line pattern** — real main-star coordinates converted to 3D, glowing star points + indigo lines forming the figure of the serpent-bearer
- a large number of stars scattered around it (two layers, far and near) as decoration
- page content is minimal: an extra-bold oversized `KADA` title + `short link platform` subtitle + login/register entries in the top-right corner
- mouse parallax + star animation give the "3D" a sense of depth
- color key: deep-space blue-purple gradient + brand indigo

## approach

use **Three.js** (vanilla, without react-three-fiber) in a client component to render the scene.

## scope

- `frontend/package.json`: add `three`, `@types/three`
- new `frontend/src/components/StarfieldCanvas.tsx` (Three.js scene rendering, client component)
- new `frontend/src/lib/ophiuchus.ts` (Ophiuchus catalog coordinate data + RA/Dec→3D conversion)
- rewrite `frontend/src/app/page.tsx` (single-screen no-scroll layout + overlay text)
- `frontend/src/app/globals.css` (deep-space background tokens / gradient utility classes)

**out of scope**: the dashboard pages, the `r/[code]` redirect page, the login/register pages, any backend change.

## page structure (page.tsx)

```
<main className="relative h-dvh w-full overflow-hidden [deep-space background gradient]">
  <StarfieldCanvas className="absolute inset-0" />   ← canvas fills the background
  <div className="absolute inset-0 z-10 flex flex-col">
    <header> (logo) Kada      login   [register] </header>
    <div centered>
      K A D A                       ← oversized main title
      short link platform           ← subtitle
    </div>
  </div>
</main>
```

- the outer `h-dvh` (fills the visible height when the mobile browser toolbar collapses) + `overflow-hidden` guarantee no scrolling
- text overlays the canvas with `absolute inset-0`, `z-10`
- login/register entries: top-right corner, login = text link, register = small solid indigo button (reusing the existing `/login` `/register` routes)

## Three.js scene (StarfieldCanvas.tsx)

### renderer
- `WebGLRenderer`, `antialias: true`, `alpha: true` (lets the underlying CSS gradient background show through)
- `PerspectiveCamera`, fov ≈ 60, camera directly in front on the z axis, eye distance about 30~40

### starfield (two layers)
- **far layer**: about 2000 stars distributed on a large-radius spherical shell around the camera (radius about 80~120), slowly rotating as a whole
- **near layer**: about 300 stars in a volume closer to the camera, with more pronounced parallax displacement
- material: `PointsMaterial` + `vertexColors`, `sizeAttenuation: true`, `transparent` + `AdditiveBlending` + `depthWrite: false`
- colors: white, pale blue (around #a5c8ff), warm white, mixed at random

### Ophiuchus constellation
- data comes from `src/lib/ophiuchus.ts`: about 12 real main stars (α Rasalhague, β Cebalrai, γ, δ Yed Prior, ε Yed Posterior, ζ, η Sabik, θ, κ, 36, 42, 58 Oph, etc.)
- the catalog stores RA (right ascension) and Dec (declination), converted into 3D points by "RA/Dec → unit-sphere 3D coordinates → scale" and placed facing the camera, relatively centered
- rendering:
  - star points: `Points` (brighter and larger than the background stars) + a soft glow sprite per star (radial-gradient texture generated on a canvas, `Sprite`), white tinted pale blue
  - lines: `Line` connects the main stars into the serpent-bearer outline in IAU style, `LineBasicMaterial` semi-transparent indigo (#4f46e5) + additive
- the whole group floats and rotates slightly

### background atmosphere
- a CSS radial gradient (deep navy → indigo → faint purple) sits beneath the canvas
- 1~2 very faint nebula glow `Sprite`s inside the scene (radial-gradient texture generated on a canvas, additive, low opacity) placed behind the constellation to add depth

### interaction and animation
- mouse parallax: listen for pointermove, use the normalized mouse position as the camera offset target, and follow it with lerp easing inside the `requestAnimationFrame` loop (quaternion or position offset)
- stars rotate slowly / the near layer drifts slightly
- disable automatic animation and parallax under `prefers-reduced-motion`
- on resize, update the camera aspect and the renderer size together (ResizeObserver or window resize)

### lifecycle cleanup
- on component unmount, cancel the rAF, dispose of geometry / material / renderer, and remove event listeners

## visual style (globals.css)

- deep-space background gradient: implement with CSS variables or Tailwind arbitrary values, a `radial-gradient` from deep navy to indigo to faint purple
- main title: system font stack (no external fonts, the domestic network is unreliable), `font-black` + `text-5xl sm:text-7xl` + wide tracking, near-white + a very faint indigo `text-shadow` glow
- subtitle `short link platform`: Chinese system font stack, one size smaller, light indigo
- login/register reuse the existing indigo brand color token (`--color-brand`)

## error handling

- WebGL unavailable / init failure: catch it and keep the CSS gradient background + text layer, degrading the page to a plain gradient display (the copy still renders fine without the JS scene)
- no SSR-related APIs inside the component; `StarfieldCanvas` declares `"use client"`, and the canvas is only initialized after mount

## testing

- `npm run lint`, `npm run build` pass
- manual verification: single screen with no scrolling; mouse parallax works; stars rotate/drift; resize works; `h-dvh` fills the screen on mobile; reduced-motion disables animation; no console errors; graceful degradation when WebGL is unavailable
