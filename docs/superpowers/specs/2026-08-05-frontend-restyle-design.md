# 前端主页面 / 登录注册页 重构设计

日期：2026-08-05
状态：已与用户确认

## 目标

对 Kada 前端三个页面（主页面 `/`、登录 `/login`、注册 `/register`）做「降噪不换血」式重构：

- 保留现有 indigo 多色体系，去掉渐变、降低视觉噪声、统一圆角/边框/阴影
- 登录/注册统一成同一套极简外壳，并补齐表单体验缺口
- 不动后端接口

## 范围

- `frontend/src/app/page.tsx`（主页面）
- `frontend/src/app/(auth)/login/page.tsx`
- `frontend/src/app/(auth)/register/page.tsx`
- `frontend/src/app/globals.css`（设计 token）
- 新增 `frontend/src/components/auth/`：`AuthCard`、`FormField`、`PasswordInput`

**不在范围内**：dashboard 各页、`r/[code]` 重定向页、后端任何改动、首页快捷缩短工具（用户明确不需要）。

## 设计 Token（globals.css）

- 主色 `indigo-600`（#4f46e5），只用于按钮、链接、Logo、选中态
- 边框统一 `neutral-200` 发丝级；背景统一 `neutral-50`
- 圆角两级：按钮/输入框 `rounded-lg`（8px）、卡片 `rounded-xl`（12px）；去掉 `rounded-2xl`
- 阴影一级：卡片 `shadow-sm`；去掉 `shadow-lg` / `hover:shadow-lg`
- 全部移除 `bg-gradient-to-*`

## 共享组件（新增 src/components/auth/）

- **AuthCard**：白卡片外壳 + 统一留白 + 品牌 Logo + 标题 + 副标题 + 底部链接区；登录/注册共用
- **FormField**：label + 输入框 + 内联错误提示（红字 + 输入框红边）
- **PasswordInput**：密码框 + 显示/隐藏切换；登录与注册复用

## 主页面（page.tsx）

| 区块 | 现状 | 改为 |
|---|---|---|
| Header | 白/80 + backdrop-blur + 透明边框 | sticky 白底 + `neutral-200` 发丝底边；Logo + 登录(文字) + 注册(实心) |
| Hero | 紫色渐变底纹、呼吸灯徽章、渐变字标题、大阴影按钮 | 纯色底；删徽章；标题近黑 + 单个 indigo 强调词；两枚按钮去掉大阴影 |
| 特性区 | 6 卡片 + `hover:shadow-lg` | 保留卡片结构；hover 改边框加深；图标块统一 `indigo-50` 底 |
| CTA | 紫色渐变底 | 纯 `indigo-600` 实心底、白字白按钮 |
| Footer | 品牌 + 版权 | 保持极简 |

## 登录页（page.tsx）

- 套用 `AuthCard` 外壳
- 手机号/邮箱 tab 切换保留，换成分段式控件（灰底 + 白色选中块）
- 手机号 tab：手机号（校验 11 位）+ 验证码 + 「获取验证码」倒计时按钮（重绘样式，逻辑沿用）
- 邮箱 tab：邮箱 + `PasswordInput`
- 统一内联错误 + 提交加载态

## 注册页（page.tsx）

- 套用 `AuthCard` 外壳
- 字段：昵称、邮箱、密码、**确认密码（新增）**
- 校验：邮箱格式、密码 ≥6 位、两次密码一致，内联红字提示
- 注册保持邮箱方式（后端无手机号注册接口）

## 错误处理

- 表单内联校验错误 + 提交失败的 toast（沿用现有 sonner 用法）
- 按钮加载态沿用现有 `loading` 状态模式

## 测试

- `npm run lint`、`npm run build` 通过
- 手动验证：三页面渲染、登录/注册表单交互、密码显隐、倒计时、错误提示
