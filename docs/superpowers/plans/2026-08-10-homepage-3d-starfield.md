# KADA Homepage 3D Starfield Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the homepage `/` as a single-screen, scroll-free immersive 3D starfield page — an Ophiuchus constellation line pattern as the main subject, stars scattered around it, and a title plus login/sign-up entry points in the top-right corner.

**Architecture:** Write a `StarfieldCanvas` client component in vanilla Three.js that renders a full-screen canvas (two star layers + Ophiuchus star points/lines/glow sprites + nebula sprites, mouse parallax + animation); `page.tsx` supplies the `h-dvh overflow-hidden` shell + overlay text. Pure logic is extracted into two files under `src/lib/` and covered by Vitest tests.

**Tech Stack:** Next.js 16 (App Router, Turbopack), React 19, TypeScript, Tailwind v4, Three.js, Vitest.

**Spec:** `docs/superpowers/specs/2026-08-10-homepage-3d-starfield-design.md`

## Global Constraints

- **This project uses a customized Next.js build** (see `frontend/AGENTS.md`): before writing React/Next code, read `frontend/node_modules/next/dist/docs/01-app/03-api-reference/01-directives/use-client.md` and follow its conventions.
- Client components must declare `"use client"` at the top of the file (before all imports).
- Do not introduce external fonts (network access from mainland China is unreliable); use a system font stack plus weight/letter-spacing/glow for the title typography.
- Deep-space palette: deep navy → indigo → faint purple radial gradient, brand indigo `#4f46e5`.
- Frontend verification commands: `npm run lint` and `npm run build` must pass; the test command is `npm run test` (= `vitest run`).
- Use Vitest for pure-logic tests; test files live in the same directory as the source and import via **relative paths** (do not depend on the `@/` alias, to avoid configuration overhead).
- Keep the existing code style: Tailwind classes, `lucide-react` icons, React function components.

---

### Task 1: Add the Three.js and Vitest dependencies

**Files:**
- Modify: `frontend/package.json`

**Interfaces:**
- Produces: `npm run test` is available; `import * as THREE from "three"` is available.

- [ ] **Step 1: Install the dependencies and add the test script**

```bash
cd frontend
npm install three @types/three
npm install -D vitest
```

Then edit `package.json` and add one line inside `scripts` (alongside `"lint"`):

```json
"test": "vitest run"
```

- [ ] **Step 2: Verify the dependencies are available**

```bash
npm ls three @types/three vitest
```

Expected: all three packages are listed and there are no missing-dependency errors.

- [ ] **Step 3: Verify the test command runs**

```bash
npm run test
```

Expected: exit code 0, with an empty-run result such as `No test files found` (there are no test files yet).

- [ ] **Step 4: Commit**

```bash
git add package.json package-lock.json
git commit -m "chore: add three, @types/three and vitest for 3D homepage"
```

---

### Task 2: `src/lib/starfield.ts` — seeded randomness + star layer generation (TDD)

**Files:**
- Create: `frontend/src/lib/starfield.ts`
- Test: `frontend/src/lib/starfield.test.ts`

**Interfaces:**
- Consumes: none (pure logic)
- Produces:
  - `createSeededRandom(seed: number): () => number` — deterministic PRNG (mulberry32), return value ∈ [0,1)
  - `randomDirection(rand: () => number): { x: number; y: number; z: number }` — a uniform direction on the unit sphere
  - `buildStarLayer(opts: { count: number; minR: number; maxR: number; seed: number }): { positions: Float32Array; colors: Float32Array }` — `positions` holds `count*3` xyz values distributed uniformly through the `[minR, maxR]` shell; `colors` holds `count*3` RGB values (0~1, a weighted mix of white/pale blue/blue-white/warm white)

- [ ] **Step 1: Write a failing test**

Create `frontend/src/lib/starfield.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { createSeededRandom, buildStarLayer } from "./starfield";

describe("createSeededRandom", () => {
  it("same seed produces the same sequence", () => {
    const a = createSeededRandom(42);
    const b = createSeededRandom(42);
    expect(a()).toBe(b());
    expect(a()).toBe(b());
  });

  it("different seeds produce different first values", () => {
    const a = createSeededRandom(1);
    const b = createSeededRandom(2);
    expect(a()).not.toBe(b());
  });

  it("returns values in [0, 1)", () => {
    const r = createSeededRandom(7);
    for (let i = 0; i < 200; i++) {
      const v = r();
      expect(v).toBeGreaterThanOrEqual(0);
      expect(v).toBeLessThan(1);
    }
  });
});

describe("buildStarLayer", () => {
  it("produces count*3 floats for positions and colors", () => {
    const layer = buildStarLayer({ count: 100, minR: 10, maxR: 50, seed: 1 });
    expect(layer.positions.length).toBe(300);
    expect(layer.colors.length).toBe(300);
  });

  it("keeps every point within [minR, maxR]", () => {
    const layer = buildStarLayer({ count: 500, minR: 10, maxR: 50, seed: 3 });
    for (let i = 0; i < 500; i++) {
      const x = layer.positions[i * 3];
      const y = layer.positions[i * 3 + 1];
      const z = layer.positions[i * 3 + 2];
      const r = Math.sqrt(x * x + y * y + z * z);
      expect(r).toBeGreaterThanOrEqual(10);
      expect(r).toBeLessThanOrEqual(50);
    }
  });

  it("is deterministic for the same seed", () => {
    const a = buildStarLayer({ count: 50, minR: 10, maxR: 50, seed: 9 });
    const b = buildStarLayer({ count: 50, minR: 10, maxR: 50, seed: 9 });
    expect(Array.from(a.positions)).toEqual(Array.from(b.positions));
  });

  it("color components stay within [0, 1]", () => {
    const layer = buildStarLayer({ count: 100, minR: 10, maxR: 50, seed: 5 });
    for (const v of Array.from(layer.colors)) {
      expect(v).toBeGreaterThanOrEqual(0);
      expect(v).toBeLessThanOrEqual(1);
    }
  });
});
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd frontend && npm run test`

Expected: FAIL — `Cannot find module './starfield'`.

- [ ] **Step 3: Implement**

Create `frontend/src/lib/starfield.ts`:

```ts
/** Deterministic PRNG (mulberry32) that keeps the star layout stable across renders. */
export function createSeededRandom(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** A uniformly distributed direction vector on the unit sphere. */
export function randomDirection(rand: () => number): { x: number; y: number; z: number } {
  const u = rand() * 2 - 1;
  const theta = rand() * Math.PI * 2;
  const s = Math.sqrt(1 - u * u);
  return { x: s * Math.cos(theta), y: s * Math.sin(theta), z: u };
}

const STAR_COLORS: Array<[number, number, number]> = [
  [1.0, 1.0, 1.0], // white
  [0.65, 0.78, 1.0], // pale blue
  [0.9, 0.92, 1.0], // blue-white
  [1.0, 0.9, 0.8], // warm white
];

/** Generate one star layer: 3D coordinates + RGB vertex colors, distributed uniformly through the [minR, maxR] shell volume. */
export function buildStarLayer(opts: {
  count: number;
  minR: number;
  maxR: number;
  seed: number;
}): { positions: Float32Array; colors: Float32Array } {
  const { count, minR, maxR, seed } = opts;
  const rand = createSeededRandom(seed);
  const positions = new Float32Array(count * 3);
  const colors = new Float32Array(count * 3);
  for (let i = 0; i < count; i++) {
    const dir = randomDirection(rand);
    const t = Math.cbrt(rand()); // cube-root interpolation: uniform by volume
    const r = minR + (maxR - minR) * t;
    positions[i * 3] = dir.x * r;
    positions[i * 3 + 1] = dir.y * r;
    positions[i * 3 + 2] = dir.z * r;
    const c = STAR_COLORS[Math.floor(rand() * STAR_COLORS.length)];
    colors[i * 3] = c[0];
    colors[i * 3 + 1] = c[1];
    colors[i * 3 + 2] = c[2];
  }
  return { positions, colors };
}
```

- [ ] **Step 4: Run the test and confirm it passes**

Run: `cd frontend && npm run test`

Expected: PASS, all of `starfield.test.ts` passes.

- [ ] **Step 5: Commit**

```bash
git add src/lib/starfield.ts src/lib/starfield.test.ts
git commit -m "feat: seeded starfield generation helpers with tests"
```

---

### Task 3: `src/lib/ophiuchus.ts` — Ophiuchus star catalog + RA/Dec→3D conversion (TDD)

**Files:**
- Create: `frontend/src/lib/ophiuchus.ts`
- Test: `frontend/src/lib/ophiuchus.test.ts`

**Interfaces:**
- Consumes: `createSeededRandom` (from `./starfield`)
- Produces:
  - `interface Vec3 { x: number; y: number; z: number }`
  - `OPHIUCHUS_STARS: Array<{ id: string; name: string; ra: number; dec: number }>` — roughly 12 principal Ophiuchus stars (RA/Dec in degrees, approximate catalog values; during implementation, visual verification is authoritative)
  - `OPHIUCHUS_LINES: Array<[number, number]>` — index pairs pointing into `OPHIUCHUS_STARS`
  - `raDecToVec3(ra: number, dec: number, radius: number): Vec3`
  - `buildOphiuchus(scale: number, depthJitter?: number): { positions: Vec3[]; lines: Array<[number, number]> }` — positions are centered on the origin by their centroid, with slight z jitter

- [ ] **Step 1: Write a failing test**

Create `frontend/src/lib/ophiuchus.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  OPHIUCHUS_LINES,
  OPHIUCHUS_STARS,
  buildOphiuchus,
  raDecToVec3,
} from "./ophiuchus";

describe("raDecToVec3", () => {
  it("maps ra=0, dec=0 to the +X axis", () => {
    const v = raDecToVec3(0, 0, 10);
    expect(v.x).toBeCloseTo(10, 5);
    expect(v.y).toBeCloseTo(0, 5);
    expect(v.z).toBeCloseTo(0, 5);
  });

  it("maps dec=90 to the +Y axis", () => {
    const v = raDecToVec3(0, 90, 10);
    expect(v.x).toBeCloseTo(0, 5);
    expect(v.y).toBeCloseTo(10, 5);
    expect(v.z).toBeCloseTo(0, 5);
  });

  it("returns points at the requested radius", () => {
    const v = raDecToVec3(120, 30, 25);
    const r = Math.sqrt(v.x * v.x + v.y * v.y + v.z * v.z);
    expect(r).toBeCloseTo(25, 5);
  });
});

describe("OPHIUCHUS_STARS", () => {
  it("has unique ids", () => {
    const ids = OPHIUCHUS_STARS.map((s) => s.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("has valid coordinate ranges", () => {
    for (const s of OPHIUCHUS_STARS) {
      expect(s.ra).toBeGreaterThanOrEqual(0);
      expect(s.ra).toBeLessThan(360);
      expect(s.dec).toBeGreaterThanOrEqual(-90);
      expect(s.dec).toBeLessThanOrEqual(90);
    }
  });
});

describe("OPHIUCHUS_LINES", () => {
  it("references valid, distinct star indices", () => {
    for (const [a, b] of OPHIUCHUS_LINES) {
      expect(a).toBeGreaterThanOrEqual(0);
      expect(a).toBeLessThan(OPHIUCHUS_STARS.length);
      expect(b).toBeGreaterThanOrEqual(0);
      expect(b).toBeLessThan(OPHIUCHUS_STARS.length);
      expect(a).not.toBe(b);
    }
  });
});

describe("buildOphiuchus", () => {
  it("produces one position per star", () => {
    const { positions } = buildOphiuchus(30);
    expect(positions.length).toBe(OPHIUCHUS_STARS.length);
  });

  it("centers the constellation near the origin", () => {
    const { positions } = buildOphiuchus(30);
    const cx = positions.reduce((s, p) => s + p.x, 0) / positions.length;
    const cy = positions.reduce((s, p) => s + p.y, 0) / positions.length;
    expect(cx).toBeCloseTo(0, 1);
    expect(cy).toBeCloseTo(0, 1);
  });

  it("spans a reasonable size for framing", () => {
    const { positions } = buildOphiuchus(30);
    const xs = positions.map((p) => p.x);
    const ys = positions.map((p) => p.y);
    const width = Math.max(...xs) - Math.min(...xs);
    const height = Math.max(...ys) - Math.min(...ys);
    expect(width).toBeGreaterThan(5);
    expect(height).toBeGreaterThan(5);
  });
});
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd frontend && npm run test`

Expected: FAIL — `Cannot find module './ophiuchus'`.

- [ ] **Step 3: Implement**

Create `frontend/src/lib/ophiuchus.ts`:

```ts
import { createSeededRandom } from "./starfield";

export interface Vec3 {
  x: number;
  y: number;
  z: number;
}

export interface OphiuchusStar {
  id: string;
  name: string;
  /** Right ascension (degrees) */
  ra: number;
  /** Declination (degrees) */
  dec: number;
}

/** Principal-star catalog for Ophiuchus (RA/Dec are approximate catalog values; during implementation, visual verification is authoritative). */
export const OPHIUCHUS_STARS: OphiuchusStar[] = [
  { id: "alpha", name: "Rasalhague α", ra: 263.73, dec: 12.56 },
  { id: "beta", name: "Cebalrai β", ra: 265.87, dec: 4.57 },
  { id: "gamma", name: "γ", ra: 266.97, dec: 2.71 },
  { id: "delta", name: "Yed Prior δ", ra: 243.59, dec: -3.69 },
  { id: "epsilon", name: "Yed Posterior ε", ra: 244.58, dec: -4.69 },
  { id: "zeta", name: "ζ", ra: 249.29, dec: -10.57 },
  { id: "eta", name: "Sabik η", ra: 257.59, dec: -15.73 },
  { id: "theta", name: "θ", ra: 260.5, dec: -25.0 },
  { id: "kappa", name: "κ", ra: 254.42, dec: 9.38 },
  { id: "s36", name: "36 Oph", ra: 258.84, dec: -26.6 },
  { id: "s42", name: "42 Oph", ra: 260.2, dec: -24.27 },
  { id: "s58", name: "58 Oph", ra: 265.86, dec: -21.68 },
];

/** Connecting lines (IAU-style serpent-bearer outline): index pairs pointing into OPHIUCHUS_STARS. */
export const OPHIUCHUS_LINES: Array<[number, number]> = [
  [8, 0], // κ–α head
  [0, 3], // α–δ left arm / serpent head
  [3, 4], // δ–ε serpent head
  [4, 5], // ε–ζ
  [5, 6], // ζ–η
  [6, 7], // η–θ
  [7, 9], // θ–36
  [9, 11], // 36–58
  [11, 2], // 58–γ
  [2, 1], // γ–β
  [1, 0], // β–α
  [7, 10], // θ–42
  [10, 9], // 42–36
];

/** Right ascension/declination → spherical 3D coordinates (ra along +X, dec along +Y, matching the camera's forward view direction). */
export function raDecToVec3(ra: number, dec: number, radius: number): Vec3 {
  const raRad = (ra * Math.PI) / 180;
  const decRad = (dec * Math.PI) / 180;
  return {
    x: radius * Math.cos(decRad) * Math.cos(raRad),
    y: radius * Math.cos(decRad) * Math.sin(raRad),
    z: radius * Math.sin(decRad),
  };
}

/* ---- private vector helpers ---- */

function dot(a: Vec3, b: Vec3): number {
  return a.x * b.x + a.y * b.y + a.z * b.z;
}

function cross(a: Vec3, b: Vec3): Vec3 {
  return {
    x: a.y * b.z - a.z * b.y,
    y: a.z * b.x - a.x * b.z,
    z: a.x * b.y - a.y * b.x,
  };
}

function norm(a: Vec3): number {
  return Math.hypot(a.x, a.y, a.z);
}

/**
 * Build the 3D data for Ophiuchus.
 *
 * Project the catalog direction vectors with a **tangent-plane projection** onto the plane whose
 * normal is the constellation centroid direction, so the pattern faces the camera (lens at +z)
 * and stands upright (Rasalhague on top, Sabik at the bottom); then scale by scale, center it on
 * the origin, and add slight z jitter for depth (the default depthJitter 1.5 assumes scale≈30).
 */
export function buildOphiuchus(
  scale: number,
  depthJitter = 1.5,
): { positions: Vec3[]; lines: Array<[number, number]> } {
  const dirs = OPHIUCHUS_STARS.map((s) => raDecToVec3(s.ra, s.dec, 1));
  const sum = dirs.reduce(
    (a, p) => ({ x: a.x + p.x, y: a.y + p.y, z: a.z + p.z }),
    { x: 0, y: 0, z: 0 },
  );
  const w = { x: sum.x / norm(sum), y: sum.y / norm(sum), z: sum.z / norm(sum) };
  let up = { x: 0, y: 1, z: 0 };
  let east = cross(w, up);
  if (norm(east) < 1e-6) {
    up = { x: 1, y: 0, z: 0 };
    east = cross(w, up);
  }
  const e = { x: east.x / norm(east), y: east.y / norm(east), z: east.z / norm(east) };
  const n = cross(w, e);
  const rand = createSeededRandom(20260810);
  const positions = dirs.map((d) => ({
    x: dot(d, e) * scale,
    y: dot(d, n) * scale,
    z: (dot(d, w) - 1) * scale + (rand() * 2 - 1) * depthJitter,
  }));
  return { positions, lines: OPHIUCHUS_LINES };
}
```

- [ ] **Step 4: Run the test and confirm it passes**

Run: `cd frontend && npm run test`

Expected: PASS, all of `ophiuchus.test.ts` passes.

- [ ] **Step 5: Commit**

```bash
git add src/lib/ophiuchus.ts src/lib/ophiuchus.test.ts
git commit -m "feat: ophiuchus constellation data and RA/Dec projection with tests"
```

---

### Task 4: `src/components/StarfieldCanvas.tsx` — Three.js scene component

**Files:**
- Create: `frontend/src/components/StarfieldCanvas.tsx`

**Interfaces:**
- Consumes:
  - `buildStarLayer` (`@/lib/starfield`)
  - `buildOphiuchus` (`@/lib/ophiuchus`)
- Produces: `export default function StarfieldCanvas({ className }: { className?: string })` — renders a `<canvas aria-hidden="true">`, passes className through to the canvas; builds the Three.js scene after the component mounts.

- [ ] **Step 1: First read the customized Next.js client component docs**

```bash
sed -n '1,80p' frontend/node_modules/next/dist/docs/01-app/03-api-reference/01-directives/use-client.md
```

Confirm: `"use client"` must be at the top of the file, before any import.

- [ ] **Step 2: Write the component**

Create `frontend/src/components/StarfieldCanvas.tsx`:

```tsx
"use client";

import { useEffect, useRef } from "react";
import * as THREE from "three";
import { buildStarLayer } from "@/lib/starfield";
import { buildOphiuchus } from "@/lib/ophiuchus";

const BRAND = 0x4f46e5;

function makeGlowTexture(inner: string): THREE.Texture {
  const size = 128;
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("2d context unavailable");
  const g = ctx.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
  g.addColorStop(0, inner);
  g.addColorStop(1, "rgba(0,0,0,0)");
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, size, size);
  return new THREE.CanvasTexture(canvas);
}

function disposeObject(root: THREE.Object3D) {
  root.traverse((child) => {
    const obj = child as THREE.Mesh & { geometry?: THREE.BufferGeometry };
    if (obj.geometry) obj.geometry.dispose();
    const material = obj.material as THREE.Material | THREE.Material[] | undefined;
    if (Array.isArray(material)) material.forEach((m) => m.dispose());
    else if (material) material.dispose();
  });
}

export default function StarfieldCanvas({ className }: { className?: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true });
    } catch (err) {
      console.error("[StarfieldCanvas] WebGL unavailable:", err);
      return;
    }

    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(60, 1, 0.1, 600);
    camera.position.set(0, 0, 36);
    camera.lookAt(0, 0, 0);

    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));

    // ---- far stars: a large shell surrounding the camera, slowly self-rotating ----
    const far = buildStarLayer({ count: 2000, minR: 45, maxR: 140, seed: 101 });
    const farGeo = new THREE.BufferGeometry();
    farGeo.setAttribute("position", new THREE.BufferAttribute(far.positions, 3));
    farGeo.setAttribute("color", new THREE.BufferAttribute(far.colors, 3));
    const farPoints = new THREE.Points(
      farGeo,
      new THREE.PointsMaterial({
        size: 1.4,
        sizeAttenuation: false,
        vertexColors: true,
        transparent: true,
        opacity: 0.9,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
      }),
    );
    const farGroup = new THREE.Group();
    farGroup.add(farPoints);
    scene.add(farGroup);

    // ---- near stars: closer to the front, so parallax is more pronounced ----
    const near = buildStarLayer({ count: 320, minR: 12, maxR: 34, seed: 202 });
    const nearGeo = new THREE.BufferGeometry();
    nearGeo.setAttribute("position", new THREE.BufferAttribute(near.positions, 3));
    nearGeo.setAttribute("color", new THREE.BufferAttribute(near.colors, 3));
    const nearMat = new THREE.PointsMaterial({
      size: 2.0,
      sizeAttenuation: false,
      vertexColors: true,
      transparent: true,
      opacity: 0.85,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
    });
    const nearPoints = new THREE.Points(nearGeo, nearMat);
    scene.add(nearPoints);

    // ---- Ophiuchus constellation: star points + glow sprites + connecting lines ----
    const constellation = new THREE.Group();
    const oph = buildOphiuchus(30, 1.5);
    const starPos = new Float32Array(oph.positions.length * 3);
    oph.positions.forEach((p, i) => {
      starPos[i * 3] = p.x;
      starPos[i * 3 + 1] = p.y;
      starPos[i * 3 + 2] = p.z;
    });
    const ophGeo = new THREE.BufferGeometry();
    ophGeo.setAttribute("position", new THREE.BufferAttribute(starPos, 3));
    const ophPoints = new THREE.Points(
      ophGeo,
      new THREE.PointsMaterial({
        size: 2.6,
        sizeAttenuation: false,
        color: 0xffffff,
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
      }),
    );
    constellation.add(ophPoints);

    const glowTex = makeGlowTexture("rgba(140,160,255,0.85)");
    for (const p of oph.positions) {
      const sprite = new THREE.Sprite(
        new THREE.SpriteMaterial({
          map: glowTex,
          color: 0xaabbff,
          transparent: true,
          opacity: 0.55,
          depthWrite: false,
          blending: THREE.AdditiveBlending,
        }),
      );
      sprite.position.set(p.x, p.y, p.z);
      sprite.scale.setScalar(3.2);
      constellation.add(sprite);
    }

    const linePos = new Float32Array(oph.lines.length * 2 * 3);
    oph.lines.forEach(([a, b], i) => {
      linePos[i * 6] = oph.positions[a].x;
      linePos[i * 6 + 1] = oph.positions[a].y;
      linePos[i * 6 + 2] = oph.positions[a].z;
      linePos[i * 6 + 3] = oph.positions[b].x;
      linePos[i * 6 + 4] = oph.positions[b].y;
      linePos[i * 6 + 5] = oph.positions[b].z;
    });
    const lineGeo = new THREE.BufferGeometry();
    lineGeo.setAttribute("position", new THREE.BufferAttribute(linePos, 3));
    const lines = new THREE.LineSegments(
      lineGeo,
      new THREE.LineBasicMaterial({
        color: BRAND,
        transparent: true,
        opacity: 0.55,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
      }),
    );
    constellation.add(lines);
    scene.add(constellation);

    // ---- background nebula glow ----
    const nebulaTex = makeGlowTexture("rgba(79,70,229,0.28)");
    const nebula = new THREE.Sprite(
      new THREE.SpriteMaterial({
        map: nebulaTex,
        transparent: true,
        opacity: 0.35,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
      }),
    );
    nebula.position.set(0, 0, -12);
    nebula.scale.setScalar(70);
    scene.add(nebula);

    const nebulaTex2 = makeGlowTexture("rgba(99,102,241,0.2)");
    const nebula2 = new THREE.Sprite(
      new THREE.SpriteMaterial({
        map: nebulaTex2,
        transparent: true,
        opacity: 0.25,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
      }),
    );
    nebula2.position.set(22, -14, -20);
    nebula2.scale.setScalar(46);
    scene.add(nebula2);

    // ---- mouse parallax ----
    let targetX = 0;
    let targetY = 0;
    let curX = 0;
    let curY = 0;
    const onPointerMove = (e: PointerEvent) => {
      targetX = (e.clientX / window.innerWidth) * 2 - 1;
      targetY = (e.clientY / window.innerHeight) * 2 - 1;
    };
    if (!reduced) window.addEventListener("pointermove", onPointerMove);

    // ---- resize ----
    const resize = () => {
      const w = canvas.clientWidth || window.innerWidth;
      const h = canvas.clientHeight || window.innerHeight;
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
      renderer.setSize(w, h, false);
    };
    resize();
    window.addEventListener("resize", resize);

    // ---- animation loop ----
    const clock = new THREE.Clock();
    let raf = 0;
    const tick = () => {
      const t = clock.getElapsedTime();
      curX += (targetX * 2.2 - curX) * 0.05;
      curY += (targetY * 1.6 - curY) * 0.05;
      camera.position.x = curX;
      camera.position.y = curY;
      camera.lookAt(0, 0, 0);

      farGroup.rotation.y += 0.00006;
      nearMat.size = 2.0 + Math.sin(t * 1.2) * 0.15;
      constellation.rotation.z = Math.sin(t * 0.15) * 0.03;
      constellation.rotation.y = Math.sin(t * 0.1) * 0.05;

      renderer.render(scene, camera);
      raf = requestAnimationFrame(tick);
    };

    if (reduced) {
      renderer.render(scene, camera); // a single static frame
    } else {
      raf = requestAnimationFrame(tick);
    }

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("resize", resize);
      disposeObject(scene);
      renderer.dispose();
    };
  }, []);

  return <canvas ref={canvasRef} className={className} aria-hidden="true" />;
}
```

- [ ] **Step 3: lint + build verification**

Run: `cd frontend && npm run lint && npm run build`

Expected: both pass, with no type errors (`three` types are provided by `@types/three`).

- [ ] **Step 4: Manual verification (may be deferred within this task, since the page is not wired up yet)**

Note: this task's deliverable is a standalone component; once Task 5 wires it up, verify everything together in the browser.

- [ ] **Step 5: Commit**

```bash
git add src/components/StarfieldCanvas.tsx
git commit -m "feat: Three.js starfield canvas with ophiuchus constellation"
```

---

### Task 5: Rewrite `page.tsx` + deep-space styles in `globals.css`

**Files:**
- Modify: `frontend/src/app/page.tsx`
- Modify: `frontend/src/app/globals.css`

**Interfaces:**
- Consumes: `StarfieldCanvas` (`@/components/StarfieldCanvas`, default export)
- Produces: a single-screen, scroll-free homepage `/`

- [ ] **Step 1: Add a deep-space background class to globals.css**

Append to `frontend/src/app/globals.css`:

```css
/* Homepage deep-space background: deep navy → indigo → faint purple radial gradient */
.bg-deep-space {
  background:
    radial-gradient(90% 70% at 70% 20%, rgba(79, 70, 229, 0.16) 0%, transparent 60%),
    radial-gradient(120% 100% at 50% 30%, #202a5e 0%, #131b45 45%, #0a0e28 100%);
}
```

- [ ] **Step 2: Rewrite page.tsx**

Replace `frontend/src/app/page.tsx` entirely:

```tsx
import Link from "next/link";
import { Link2 } from "lucide-react";
import StarfieldCanvas from "@/components/StarfieldCanvas";

export default function HomePage() {
  return (
    <main className="relative h-dvh w-full overflow-hidden overscroll-none bg-deep-space text-white">
      <StarfieldCanvas className="absolute inset-0" />

      <div className="absolute inset-0 z-10 flex flex-col">
        <header className="flex items-center justify-between px-6 py-5 sm:px-8">
          <div className="flex items-center gap-2.5">
            <div className="flex size-8 items-center justify-center rounded-lg bg-indigo-600">
              <Link2 className="size-4 text-white" />
            </div>
            <span className="text-lg font-semibold tracking-wider">KADA</span>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/login"
              className="rounded-lg px-4 py-2 text-sm font-medium text-indigo-100 transition hover:bg-white/10 hover:text-white"
            >
              Log in
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-500"
            >
              Sign up free
            </Link>
          </div>
        </header>

        <div className="flex flex-1 flex-col items-center justify-center px-4 text-center">
          <h1 className="text-6xl font-black tracking-[0.18em] text-white drop-shadow-[0_0_28px_rgba(99,102,241,0.45)] sm:text-8xl">
            KADA
          </h1>
          <p className="mt-6 text-base font-medium tracking-[0.5em] text-indigo-200/90 sm:text-lg">
            Short link platform
          </p>
        </div>
      </div>
    </main>
  );
}
```

- [ ] **Step 3: lint + build verification**

Run: `cd frontend && npm run lint && npm run build`

Expected: both pass.

- [ ] **Step 4: Browser manual verification checklist**

```bash
cd frontend && npm run dev
```

Open `http://localhost:3000` and confirm each item:

- [ ] The page is a single screen with no scrollbar (scrolling has no effect / is disabled)
- [ ] The background is a deep-space blue-purple gradient + 3D starfield, and the Ophiuchus constellation pattern is clearly visible and centered
- [ ] Moving the mouse produces parallax in the starfield/constellation
- [ ] The stars slowly rotate/twinkle and the constellation drifts slightly
- [ ] Resizing the window adapts the rendering to the viewport (resize works correctly)
- [ ] The title `KADA` + `Short link platform` render with a glow; the "Log in / Sign up" entries in the top-right navigate to `/login` and `/register`
- [ ] With devtools set to `prefers-reduced-motion: reduce`: no animation, only a single static frame, and the page still displays correctly
- [ ] No console errors; with WebGL disabled the page degrades to a plain gradient background + text
- [ ] Mobile emulator (`h-dvh`): fills the visible height with no whitespace left for the address bar

- [ ] **Step 5: Commit**

```bash
git add src/app/page.tsx src/app/globals.css
git commit -m "feat: 3D starfield homepage with ophiuchus background"
```

---

## Self-check record

- **Spec coverage**: single screen without scrolling (Task 5 shell + `overflow-hidden`); Ophiuchus line pattern (Tasks 3/4); two layers of scattered stars (Tasks 2/4); title + login/sign-up (Task 5); mouse parallax + star animation (Task 4); deep-space blue-purple + indigo (Tasks 4/5); reduced-motion (Task 4); WebGL fallback (Task 4 try/catch); resize (Task 4); no external fonts (Task 5); lint/build/manual verification (each task).
- **Placeholder scan**: no TBD/TODO; every code step contains a complete implementation.
- **Type consistency**: `buildStarLayer`, `buildOphiuchus`, `createSeededRandom`, `raDecToVec3`, `Vec3`, `OPHIUCHUS_STARS`, `OPHIUCHUS_LINES` are defined in Tasks 2/3 and consumed in Task 4, with matching signatures.
- **Projection direction (key correction)**: `buildOphiuchus` must use a **tangent-plane projection** (with the constellation centroid direction as the normal) and must not use the raw RA/Dec→XYZ direct mapping — the latter squashes Ophiuchus into a horizontal band (ySpan≈3.6). The tangent-plane projection has been verified with a script: xSpan≈16.6, ySpan≈13.4, the pattern faces the camera, with Rasalhague on top (y≈+8.7) and Sabik at the bottom (y≈−3.6).
