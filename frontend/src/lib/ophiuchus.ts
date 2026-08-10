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
    y: radius * Math.sin(decRad),
    z: radius * Math.cos(decRad) * Math.sin(raRad),
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
  const n = cross(e, w);
  const rand = createSeededRandom(20260810);
  const positions = dirs.map((d) => ({
    x: dot(d, e) * scale,
    y: dot(d, n) * scale,
    z: (dot(d, w) - 1) * scale + (rand() * 2 - 1) * depthJitter,
  }));
  return { positions, lines: OPHIUCHUS_LINES.slice() };
}
