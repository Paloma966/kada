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
    // 每颗星随机亮度（0.35~1.0），让星星有明暗层次
    const brightness = 0.35 + rand() * 0.65;
    colors[i * 3] = c[0] * brightness;
    colors[i * 3 + 1] = c[1] * brightness;
    colors[i * 3 + 2] = c[2] * brightness;
  }
  return { positions, colors };
}
