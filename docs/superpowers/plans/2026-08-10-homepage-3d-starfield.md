# KADA 首页 3D 星空重构 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把首页 `/` 重构成单屏无滚动的沉浸式 3D 星空页——蛇夫座星座连线图案为主体，周围散布星星，标题 + 右上角登录/注册入口。

**Architecture:** 用原生 Three.js 写一个 `StarfieldCanvas` 客户端组件渲染全屏 canvas（星星两层 + 蛇夫座星点/连线/辉光精灵 + 星云精灵，鼠标视差 + 动画），`page.tsx` 提供 `h-dvh overflow-hidden` 外壳 + 覆盖层文字。纯逻辑抽到 `src/lib/` 两个文件并用 Vitest 测试。

**Tech Stack:** Next.js 16 (App Router, Turbopack), React 19, TypeScript, Tailwind v4, Three.js, Vitest。

**Spec:** `docs/superpowers/specs/2026-08-10-homepage-3d-starfield-design.md`

## Global Constraints

- **本项目是定制版 Next.js**（见 `frontend/AGENTS.md`）：写 React/Next 代码前先读 `frontend/node_modules/next/dist/docs/01-app/03-api-reference/01-directives/use-client.md`，遵守其约定。
- 客户端组件在文件顶部（所有 import 之前）声明 `"use client"`。
- 不引入外部字体（国内网络不稳定）；标题排版用系统字体栈 + 字重/字距/辉光。
- 深空配色：深 navy → 靛蓝 → 微紫径向渐变，品牌 indigo `#4f46e5`。
- 前端验证命令：`npm run lint` 与 `npm run build` 必须通过；测试命令 `npm run test`（= `vitest run`）。
- 纯逻辑测试用 Vitest，测试文件与源码同目录，用**相对路径**导入（不依赖 `@/` 别名，避免配置成本）。
- 保持既有代码风格：Tailwind 类、`lucide-react` 图标、React 函数组件。

---

### Task 1: 添加 Three.js 与 Vitest 依赖

**Files:**
- Modify: `frontend/package.json`

**Interfaces:**
- Produces: `npm run test` 可用；`import * as THREE from "three"` 可用。

- [ ] **Step 1: 安装依赖并加测试脚本**

```bash
cd frontend
npm install three @types/three
npm install -D vitest
```

然后编辑 `package.json`，在 `scripts` 里加一行（与 `"lint"` 并列）：

```json
"test": "vitest run"
```

- [ ] **Step 2: 验证依赖可用**

```bash
npm ls three @types/three vitest
```

Expected: 三个包都列出、无缺失依赖报错。

- [ ] **Step 3: 验证 test 命令可运行**

```bash
npm run test
```

Expected: 退出码 0，输出 `No test files found` 之类的空跑结果（尚无测试文件）。

- [ ] **Step 4: 提交**

```bash
git add package.json package-lock.json
git commit -m "chore: add three, @types/three and vitest for 3D homepage"
```

---

### Task 2: `src/lib/starfield.ts` — 种子随机 + 星星图层生成（TDD）

**Files:**
- Create: `frontend/src/lib/starfield.ts`
- Test: `frontend/src/lib/starfield.test.ts`

**Interfaces:**
- Consumes: 无（纯逻辑）
- Produces:
  - `createSeededRandom(seed: number): () => number` — 确定性 PRNG（mulberry32），返回值 ∈ [0,1)
  - `randomDirection(rand: () => number): { x: number; y: number; z: number }` — 单位球面上均匀方向
  - `buildStarLayer(opts: { count: number; minR: number; maxR: number; seed: number }): { positions: Float32Array; colors: Float32Array }` — `positions` 为 `count*3` 个 xyz，均匀分布在 `[minR, maxR]` 壳层；`colors` 为 `count*3` 个 RGB（0~1，白色/淡蓝/蓝白/暖白加权混用）

- [ ] **Step 1: 写失败测试**

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

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test`

Expected: FAIL — `Cannot find module './starfield'`。

- [ ] **Step 3: 实现**

Create `frontend/src/lib/starfield.ts`:

```ts
/** 确定性 PRNG（mulberry32），保证星图布局在多次渲染间稳定。 */
export function createSeededRandom(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** 单位球面上均匀分布的方向向量。 */
export function randomDirection(rand: () => number): { x: number; y: number; z: number } {
  const u = rand() * 2 - 1;
  const theta = rand() * Math.PI * 2;
  const s = Math.sqrt(1 - u * u);
  return { x: s * Math.cos(theta), y: s * Math.sin(theta), z: u };
}

const STAR_COLORS: Array<[number, number, number]> = [
  [1.0, 1.0, 1.0], // 白
  [0.65, 0.78, 1.0], // 淡蓝
  [0.9, 0.92, 1.0], // 蓝白
  [1.0, 0.9, 0.8], // 暖白
];

/** 生成一层星星：3D 坐标 + RGB 顶点色，均匀分布在 [minR, maxR] 壳层体积内。 */
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
    const t = Math.cbrt(rand()); // 立方根插值：体积均匀
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

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test`

Expected: PASS，`starfield.test.ts` 全部通过。

- [ ] **Step 5: 提交**

```bash
git add src/lib/starfield.ts src/lib/starfield.test.ts
git commit -m "feat: seeded starfield generation helpers with tests"
```

---

### Task 3: `src/lib/ophiuchus.ts` — 蛇夫座星表 + RA/Dec→3D 换算（TDD）

**Files:**
- Create: `frontend/src/lib/ophiuchus.ts`
- Test: `frontend/src/lib/ophiuchus.test.ts`

**Interfaces:**
- Consumes: `createSeededRandom`（来自 `./starfield`）
- Produces:
  - `interface Vec3 { x: number; y: number; z: number }`
  - `OPHIUCHUS_STARS: Array<{ id: string; name: string; ra: number; dec: number }>` — 约 12 颗蛇夫座主星（RA/Dec 度数，近似目录值，实现时以视觉校验为准）
  - `OPHIUCHUS_LINES: Array<[number, number]>` — 成对索引指向 `OPHIUCHUS_STARS`
  - `raDecToVec3(ra: number, dec: number, radius: number): Vec3`
  - `buildOphiuchus(scale: number, depthJitter?: number): { positions: Vec3[]; lines: Array<[number, number]> }` — 位置以质心居中于原点，带轻微 z 抖动

- [ ] **Step 1: 写失败测试**

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

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test`

Expected: FAIL — `Cannot find module './ophiuchus'`。

- [ ] **Step 3: 实现**

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
  /** 赤经（度） */
  ra: number;
  /** 赤纬（度） */
  dec: number;
}

/** 蛇夫座主星星表（RA/Dec 为近似目录值；实现时以视觉校验为准）。 */
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

/** 连线（IAU 风格持蛇者轮廓）：成对索引指向 OPHIUCHUS_STARS。 */
export const OPHIUCHUS_LINES: Array<[number, number]> = [
  [8, 0], // κ–α 头部
  [0, 3], // α–δ 左臂/蛇头
  [3, 4], // δ–ε 蛇头
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

/** 赤经/赤纬 → 球面 3D 坐标（ra 沿 +X，dec 沿 +Y，符合相机正视方向）。 */
export function raDecToVec3(ra: number, dec: number, radius: number): Vec3 {
  const raRad = (ra * Math.PI) / 180;
  const decRad = (dec * Math.PI) / 180;
  return {
    x: radius * Math.cos(decRad) * Math.cos(raRad),
    y: radius * Math.cos(decRad) * Math.sin(raRad),
    z: radius * Math.sin(decRad),
  };
}

/* ---- 私有向量工具 ---- */

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
 * 生成蛇夫座 3D 数据。
 *
 * 用**切平面投影**把星表方向向量投到以星座质心方向为法线的平面上，
 * 使图案正对相机（镜头在 +z）、上下直立（Rasalhague 在上、Sabik 在下），
 * 再缩放 scale 并居中于原点；z 带轻微抖动增加立体感。
 * depthJitter 相对 scale 应很小（默认 1.5 / scale≈30）。
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

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test`

Expected: PASS，`ophiuchus.test.ts` 全部通过。

- [ ] **Step 5: 提交**

```bash
git add src/lib/ophiuchus.ts src/lib/ophiuchus.test.ts
git commit -m "feat: ophiuchus constellation data and RA/Dec projection with tests"
```

---

### Task 4: `src/components/StarfieldCanvas.tsx` — Three.js 场景组件

**Files:**
- Create: `frontend/src/components/StarfieldCanvas.tsx`

**Interfaces:**
- Consumes:
  - `buildStarLayer`（`@/lib/starfield`）
  - `buildOphiuchus`（`@/lib/ophiuchus`）
- Produces: `export default function StarfieldCanvas({ className }: { className?: string })` — 渲染一个 `<canvas aria-hidden="true">`，className 透传给 canvas；组件挂载后构建 Three.js 场景。

- [ ] **Step 1: 先读定制版 Next.js 的客户端组件文档**

```bash
sed -n '1,80p' frontend/node_modules/next/dist/docs/01-app/03-api-reference/01-directives/use-client.md
```

Confirm: `"use client"` 必须位于文件顶部、任何 import 之前。

- [ ] **Step 2: 写组件**

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

    // ---- 远端星星：包围相机的大壳层，缓慢自转 ----
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

    // ---- 近端星星：更靠前，视差更明显 ----
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

    // ---- 蛇夫座星座：星点 + 辉光精灵 + 连线 ----
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

    // ---- 背景星云辉光 ----
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

    // ---- 鼠标视差 ----
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

    // ---- 动画循环 ----
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
      renderer.render(scene, camera); // 静态一帧
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

- [ ] **Step 3: lint + 构建验证**

Run: `cd frontend && npm run lint && npm run build`

Expected: 两者通过，无类型错误（`three` 类型已由 `@types/three` 提供）。

- [ ] **Step 4: 手动验证（本任务内可暂不触发，因为页面还没接上）**

说明：此任务交付物是独立组件；Task 5 接上后统一在浏览器验证。

- [ ] **Step 5: 提交**

```bash
git add src/components/StarfieldCanvas.tsx
git commit -m "feat: Three.js starfield canvas with ophiuchus constellation"
```

---

### Task 5: 重写 `page.tsx` + `globals.css` 深空样式

**Files:**
- Modify: `frontend/src/app/page.tsx`
- Modify: `frontend/src/app/globals.css`

**Interfaces:**
- Consumes: `StarfieldCanvas`（`@/components/StarfieldCanvas`，默认导出）
- Produces: 单屏无滚动首页 `/`

- [ ] **Step 1: globals.css 加深空背景类**

Append to `frontend/src/app/globals.css`:

```css
/* 首页深空背景：深 navy → 靛蓝 → 微紫径向渐变 */
.bg-deep-space {
  background:
    radial-gradient(90% 70% at 70% 20%, rgba(79, 70, 229, 0.16) 0%, transparent 60%),
    radial-gradient(120% 100% at 50% 30%, #202a5e 0%, #131b45 45%, #0a0e28 100%);
}
```

- [ ] **Step 2: 重写 page.tsx**

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
              登录
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-500"
            >
              免费注册
            </Link>
          </div>
        </header>

        <div className="flex flex-1 flex-col items-center justify-center px-4 text-center">
          <h1 className="text-6xl font-black tracking-[0.18em] text-white drop-shadow-[0_0_28px_rgba(99,102,241,0.45)] sm:text-8xl">
            KADA
          </h1>
          <p className="mt-6 text-base font-medium tracking-[0.5em] text-indigo-200/90 sm:text-lg">
            短链接平台
          </p>
        </div>
      </div>
    </main>
  );
}
```

- [ ] **Step 3: lint + 构建验证**

Run: `cd frontend && npm run lint && npm run build`

Expected: 两者通过。

- [ ] **Step 4: 浏览器手动验证清单**

```bash
cd frontend && npm run dev
```

打开 `http://localhost:3000`，逐项确认：

- [ ] 页面单屏、无滚动条（滚动无效/被禁用）
- [ ] 背景是深空蓝紫渐变 + 3D 星空，蛇夫座星座图案清晰可见且居中
- [ ] 移动鼠标：星空/星座产生视差
- [ ] 星星缓慢自转/闪烁，星座轻微浮动
- [ ] 窗口缩放：画面随视口自适应（resize 正常）
- [ ] 标题 `KADA` + `短链接平台` 渲染、有辉光；右上角「登录/注册」可点跳转 `/login` `/register`
- [ ] 开发者工具设为 `prefers-reduced-motion: reduce`：无动画、仅静态一帧，页面仍正常显示
- [ ] 控制台无报错；禁用 WebGL 时页面降级为纯渐变背景 + 文字
- [ ] 手机模拟器（`h-dvh`）：占满可视高度、无地址栏留白

- [ ] **Step 5: 提交**

```bash
git add src/app/page.tsx src/app/globals.css
git commit -m "feat: 3D starfield homepage with ophiuchus background"
```

---

## 自检记录

- **Spec 覆盖**：单屏无滚动（Task 5 外壳 + `overflow-hidden`）；蛇夫座连线图案（Task 3/4）；散落星星两层（Task 2/4）；标题 + 登录/注册（Task 5）；鼠标视差 + 星星动画（Task 4）；深空蓝紫 + indigo（Task 4/5）；reduced-motion（Task 4）；WebGL 降级（Task 4 try/catch）；resize（Task 4）；无外部字体（Task 5）；lint/build/手动验证（各 Task）。
- **占位符扫描**：无 TBD/TODO；代码步骤均含完整实现。
- **类型一致性**：`buildStarLayer`、`buildOphiuchus`、`createSeededRandom`、`raDecToVec3`、`Vec3`、`OPHIUCHUS_STARS`、`OPHIUCHUS_LINES` 在 Task 2/3 定义、Task 4 消费，签名一致。
- **投影方向（关键修正）**：`buildOphiuchus` 必须用**切平面投影**（以星座质心方向为法线），不能用原始 RA/Dec→XYZ 直映——后者会把蛇夫座压成水平横条（ySpan≈3.6）。已用脚本验证切平面投影：xSpan≈16.6、ySpan≈13.4，图案正对相机且 Rasalhague 在上（y≈+8.7）、Sabik 在下（y≈−3.6）。
