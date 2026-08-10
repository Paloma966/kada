# KADA 首页 3D 星空重构设计

日期：2026-08-10
状态：已与用户确认

## 目标

把首页 `/` 重构成一个**单屏、无滚动、全视口沉浸式 3D 星空页**：

- 背景主体是**蛇夫座（Ophiuchus）星座连线图案**——用真实主星坐标转 3D，发光星点 + indigo 连线成持蛇者轮廓
- 周围散布大量星星（近远两层）做点缀
- 页面内容极简：`KADA` 超粗大标题 + `短链接平台` 副标题 + 右上角登录/注册入口
- 鼠标视差 + 星星动画，让「三维」有纵深感
- 色彩基调：深空蓝紫渐变 + 品牌 indigo

## 方案

使用 **Three.js**（原生，不经 react-three-fiber）写一个客户端组件渲染场景。

## 范围

- `frontend/package.json`：新增 `three`、`@types/three`
- 新增 `frontend/src/components/StarfieldCanvas.tsx`（Three.js 场景渲染，客户端组件）
- 新增 `frontend/src/lib/ophiuchus.ts`（蛇夫座星表坐标数据 + RA/Dec→3D 换算）
- 重写 `frontend/src/app/page.tsx`（单屏无滚动布局 + 覆盖层文字）
- `frontend/src/app/globals.css`（深空背景 token / 渐变工具类）

**不在范围内**：dashboard 各页、`r/[code]` 重定向页、登录/注册页、后端任何改动。

## 页面结构（page.tsx）

```
<main className="relative h-dvh w-full overflow-hidden [深空背景渐变]">
  <StarfieldCanvas className="absolute inset-0" />   ← canvas 铺满背景
  <div className="absolute inset-0 z-10 flex flex-col">
    <header> (logo) Kada      登录   [注册] </header>
    <div 居中>
      K A D A                       ← 超大主标题
      短链接平台                     ← 副标题
    </div>
  </div>
</main>
```

- 外层 `h-dvh`（移动端收起浏览器工具栏时占满可视高度）+ `overflow-hidden` 保证无滚动
- 文字用 `absolute inset-0` 覆盖在 canvas 之上，`z-10`
- 登录/注册入口：右上角，登录=文字链接、注册=indigo 实心小按钮（沿用现有 `/login` `/register` 路由）

## Three.js 场景（StarfieldCanvas.tsx）

### 渲染器
- `WebGLRenderer`，`antialias: true`，`alpha: true`（透出底层 CSS 渐变背景）
- `PerspectiveCamera`，fov ≈ 60，相机位于 z 轴正前方，视点距离约 30~40

### 星空（两层）
- **远端层**：约 2000 颗星星，分布在相机外围大半径球壳上（半径约 80~120），缓慢整体自转
- **近端层**：约 300 颗星星，分布在更靠近相机的体积内，视差位移更明显
- 材质：`PointsMaterial` + `vertexColors`，`sizeAttenuation: true`，`transparent` + `AdditiveBlending` + `depthWrite: false`
- 颜色：白、淡蓝（#a5c8ff 附近）、暖白，随机混用

### 蛇夫座星座
- 数据来自 `src/lib/ophiuchus.ts`：约 12 颗真实主星（α Rasalhague、β Cebalrai、γ、δ Yed Prior、ε Yed Posterior、ζ、η Sabik、θ、κ、36、42、58 Oph 等）
- 星表存 RA（赤经）、Dec（赤纬），按「RA/Dec → 单位球面 3D 坐标 → 缩放」转成 3D 点，摆在面向相机、相对居中的位置上
- 渲染：
  - 星点：`Points`（比背景星更亮、更大）+ 每颗带一层软辉光精灵（canvas 生成的径向渐变贴图，`Sprite`），白色偏淡蓝
  - 连线：`Line` 按 IAU 风格把主星连成持蛇者轮廓，`LineBasicMaterial` 半透明 indigo（#4f46e5）+ additive
- 整组轻微浮动旋转

### 背景氛围
- CSS 径向渐变（深 navy → 靛蓝 → 微紫）垫在 canvas 之下
- 场景内 1~2 个极淡的星云辉光 `Sprite`（canvas 生成径向渐变贴图，additive，低透明度）放在星座后方，增强纵深

### 交互与动画
- 鼠标视差：监听 pointermove，把归一化鼠标位置作为相机偏移目标，`requestAnimationFrame` 循环里用 lerp 缓动跟随（quaternion 或位置偏置）
- 星星缓慢自转 / 近层轻微漂移
- `prefers-reduced-motion` 时关闭自动动画与视差
- resize 时同步更新相机 aspect 与 renderer 尺寸（ResizeObserver 或 window resize）

### 生命周期清理
- 组件卸载时取消 rAF、释放 geometry / material / renderer、移除事件监听

## 视觉样式（globals.css）

- 深空背景渐变：用 CSS 变量或 Tailwind arbitrary 值实现，`radial-gradient` 从深 navy 到靛蓝到微紫
- 主标题：系统字体栈（不引入外部字体，国内网络不稳），`font-black` + `text-5xl sm:text-7xl` + 大号字距（tracking），近白色 + 极淡 indigo `text-shadow` 辉光
- 副标题 `短链接平台`：中文系统栈、小一号、indigo 浅色
- 登录/注册沿用现有 indigo 主色 token（`--color-brand`）

## 错误处理

- WebGL 不可用 / 初始化失败：catch 并保留 CSS 渐变背景 + 文字层，页面降级为纯渐变展示（无 JS 场景也能正常显示文案）
- 组件内不使用 SSR 相关 API；`StarfieldCanvas` 声明 `"use client"`，canvas 只在挂载后初始化

## 测试

- `npm run lint`、`npm run build` 通过
- 手动验证：单屏无滚动；鼠标视差生效；星星自转/漂移；resize 正常；移动端 `h-dvh` 占满；reduced-motion 关闭动画；无控制台报错；WebGL 不可用时降级正常
