# 登录/注册页套用星空背景 + 手机号注册 设计

日期：2026-08-10
状态：已与用户确认

## 目标

1. 把首页的 3D 星空背景套用到登录/注册页，统一视觉
2. 修复「不能用手机号注册」：注册页增加手机号注册入口

## 范围

**前端（改 5 个文件）**
- `frontend/src/components/auth/AuthCard.tsx` — 套星空背景 + 透明玻璃卡片
- `frontend/src/components/auth/FormField.tsx` — 暗色 label / inputBase / fieldState
- `frontend/src/components/auth/PasswordInput.tsx` — 暗色输入框
- `frontend/src/app/(auth)/login/page.tsx` — 暗色 tab 控件、「获取验证码」按钮样式微调
- `frontend/src/app/(auth)/register/page.tsx` — 加手机号/邮箱 tab + 手机号注册流程

**后端：零改动**

已验证后端能力（无需改）：
- `POST /auth/login-by-phone`（`backend/internal/service/auth_service.go`）—— 验证码登录，找不到用户就自动建号（自动注册）
- `PATCH /me`（`backend/internal/handler/auth/handler.go:45`）+ `api.ts:61 updateMe(token, { name })` —— 认证后可写昵称
- dev 模式短信验证码打印在后端日志：`📱 [DEV] Phone: %s, Code: %s`

**不在范围内**：dashboard 各页、`r/[code]`、后端任何改动、首页（已完成）。

## 星空背景（AuthCard）

AuthCard 从浅色 `bg-neutral-50` 包装改为：

```tsx
<div className="relative min-h-dvh bg-deep-space">
  <StarfieldCanvas className="fixed inset-0 h-full w-full" />
  <div className="relative z-10 flex min-h-dvh items-center justify-center px-4 py-12">
    {/* logo / 标题 / 卡片 / footer */}
  </div>
</div>
```

- 复用 `bg-deep-space` 渐变 + `StarfieldCanvas`（`fixed inset-0`，滚动时背景固定）
- 卡片：**透明玻璃** `rounded-xl border border-white/10 bg-white/[0.04] p-6 backdrop-blur-lg sm:p-8`
- 标题白字；副标题 `text-indigo-200/80`；footer 链接 `text-indigo-300`
- logo 块保持 indigo 实心

## 暗色表单组件

**FormField**：
- label：`text-neutral-100`
- `inputBase`：`bg-white/10 border-transparent text-white placeholder:text-neutral-400/70 focus:ring-2`
- `fieldState`：invalid → `border-red-400/60 focus:ring-red-400/30`；正常 → `border-white/15 focus:border-indigo-400 focus:ring-indigo-400/30`
- 错误文案：`text-red-400`

**PasswordInput**：与 inputBase 同款暗色；眼睛按钮 `text-neutral-400 hover:text-neutral-200`

**login/page.tsx** 内联样式微调：
- 分段 tab 容器 `bg-neutral-100` → `bg-white/10`；未选中文字 `text-neutral-300 hover:text-white`；选中块 `bg-white text-neutral-900 shadow-sm`（保持白色选中块）
- 「获取验证码」按钮 `bg-indigo-50 text-indigo-600` → `bg-indigo-500/20 text-indigo-200 hover:bg-indigo-500/30`

## 手机号注册（register/page.tsx）

- 注册页加**手机号 / 邮箱 分段 tab**，默认手机号 tab（与登录页一致）
- **手机号 tab**：
  - 字段：昵称（可选）+ 手机号 + 验证码（带 60s 倒计时）
  - 提交：校验手机号 11 位、验证码 6 位、昵称非必填
  - 流程：
    1. `authAPI.sendSMSCode(phone)`
    2. `authAPI.loginByPhone(phone, code)` → 后端自动注册 + 返回 token/user
    3. 昵称：`const nickname = name.trim() || \`用户${phone.slice(-4)}\``
    4. 若返回的 `user.name` 为空 → `authAPI.updateMe(token, { name: nickname })` 写入昵称
    5. `setToken/setUser` + 跳 `/dashboard`
  - 已注册手机号直接登录（不覆盖已有昵称）
- **邮箱 tab**：沿用现有邮箱注册（昵称+邮箱+密码+确认），逻辑不动

## 错误处理

- 沿用现有 toast + 内联错误模式
- 发送验证码失败、验证码错误、登录失败 → toast 展示后端返回信息

## 测试

- `npm run test`（现有 18 项，预计不变）
- `npm run build` 通过
- 后端：`go test ./...`（未改动，回归确认）
- 手动验证：
  - 登录/注册页显示星空背景 + 透明玻璃卡片，文字可读
  - 手机号注册：输入手机号 → 获取验证码（dev 日志可见）→ 输入验证码+昵称 → 注册成功跳转 dashboard，昵称正确
  - 手机号登录仍正常；邮箱注册/登录仍正常
  - 小屏可滚动时星空背景固定
