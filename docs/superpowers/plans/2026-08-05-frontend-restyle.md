# 前端主页面 / 登录注册页 重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Kada 主页面 `/`、登录 `/login`、注册 `/register` 三个页面重构为统一、克制的极简风格，并补齐登录/注册表单的校验与密码显隐功能。

**Architecture:** 先在 `globals.css` 用 Tailwind v4 `@theme` 定义语义化 design tokens；新增 `src/components/auth/` 下三个共享组件（`AuthCard`、`FormField`、`PasswordInput`）；登录/注册复用该外壳，主页面按同一套 token 重排。纯前端改动，不动后端接口。

**Tech Stack:** Next.js 16.2.10 (App Router), React 19, Tailwind CSS v4, lucide-react。

**测试方式说明：** 本项目没有组件级测试框架（无 vitest/jest）。每个任务的验证周期为 `npm run lint` + `npm run build`（类型检查 + 编译），加浏览器手动冒烟。

## Global Constraints

- 后端接口一律不动（`src/lib/api.ts` 中的 API 名称/签名保持不变）
- 保留 indigo 色系，仅去掉渐变（`bg-gradient-to-*`）、去掉 `shadow-lg`/`hover:shadow-lg`、去掉 `rounded-2xl`
- 边框统一 `neutral-200` 发丝级；背景统一 `neutral-50`（页面底色）/ `white`（卡片与区块）
- 圆角两级：按钮/输入框 `rounded-lg`(8px)、卡片 `rounded-xl`(12px)
- 新增文件放到 `src/components/auth/`
- 先读 `node_modules/next/dist/docs/` 中与客户端组件/表单相关的指南，确认本改动不受 Next 16 破坏性变更影响（本项目页面均已用 `"use client"`，无需改路由结构）

---
## 文件结构

- Modify: `frontend/src/app/globals.css` — 设计 token
- Create: `frontend/src/components/auth/AuthCard.tsx`
- Create: `frontend/src/components/auth/FormField.tsx`（含 `FormField` 组件 + `inputBase` / `fieldState` 工具）
- Create: `frontend/src/components/auth/PasswordInput.tsx`
- Modify: `frontend/src/app/page.tsx` — 主页面重排
- Modify: `frontend/src/app/(auth)/login/page.tsx` — 登录页重构 + 校验
- Modify: `frontend/src/app/(auth)/register/page.tsx` — 注册页重构 + 校验

---

### Task 1: 设计 Token（globals.css）

**Files:**
- Modify: `frontend/src/app/globals.css`

**Interfaces:**
- 产出：Tailwind 语义化工具类 `bg-brand` / `text-brand` / `border-hairline` / `bg-surface`（本任务只声明，后续任务按需用；也可直接用标准 indigo/neutral 类，二者等价）

- [ ] **Step 1: 重写 globals.css**

将 `frontend/src/app/globals.css` 替换为：

```css
@import "tailwindcss";

@theme {
  --color-brand: #4f46e5;        /* indigo-600 主色 */
  --color-brand-dark: #4338ca;   /* indigo-700 */
  --color-brand-soft: #eef2ff;   /* indigo-50 */
  --color-hairline: #e5e5e5;     /* neutral-200 发丝边框 */
  --color-surface: #fafafa;      /* neutral-50 页面底色 */
}

:root {
  --background: #ffffff;
  --foreground: #171717;
}

body {
  background: var(--background);
  color: var(--foreground);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
    "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
}
```

- [ ] **Step 2: 验证编译**

Run: `cd frontend && npm run build`
Expected: 构建成功（本任务无页面改动，仅确认 token 定义不破坏 Tailwind）。

- [ ] **Step 3: Commit**

```bash
cd /home/chun/dev/projects/kada
git add frontend/src/app/globals.css
git commit -m "style: add semantic design tokens (brand/hairline/surface)"
```

---

### Task 2: 共享 Auth 组件

**Files:**
- Create: `frontend/src/components/auth/AuthCard.tsx`
- Create: `frontend/src/components/auth/FormField.tsx`
- Create: `frontend/src/components/auth/PasswordInput.tsx`

**Interfaces:**
- 产出：
  - `AuthCard({ title, subtitle, footer, children })` — 白卡片外壳
  - `FormField({ id, label, error, children })` — label + 输入框 + 内联错误
  - `inputBase` — 输入框基础类字符串
  - `fieldState(invalid: boolean)` — 输入框边框/聚焦状态类字符串
  - `PasswordInput({ id, className, ...inputProps })` — 密码框 + 显隐切换

- [ ] **Step 1: 创建 AuthCard.tsx**

```tsx
import Link from "next/link";
import { Link2 } from "lucide-react";

interface AuthCardProps {
  title: string;
  subtitle: string;
  footer: React.ReactNode;
  children: React.ReactNode;
}

export function AuthCard({ title, subtitle, footer, children }: AuthCardProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-neutral-50 px-4 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 flex flex-col items-center text-center">
          <div className="mb-4 flex size-11 items-center justify-center rounded-xl bg-indigo-600">
            <Link2 className="size-5 text-white" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight text-neutral-900">{title}</h1>
          <p className="mt-2 text-sm text-neutral-500">{subtitle}</p>
        </div>
        <div className="rounded-xl border border-neutral-200 bg-white p-6 shadow-sm sm:p-8">
          {children}
        </div>
        <p className="mt-6 text-center text-sm text-neutral-500">{footer}</p>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: 创建 FormField.tsx**

```tsx
interface FormFieldProps {
  id: string;
  label: string;
  error?: string;
  children: React.ReactNode;
}

export function FormField({ id, label, error, children }: FormFieldProps) {
  return (
    <div>
      <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-neutral-700">
        {label}
      </label>
      {children}
      {error && <p className="mt-1.5 text-xs text-red-600">{error}</p>}
    </div>
  );
}

/** 输入框基础样式（不含边框色/聚焦态） */
export const inputBase =
  "w-full rounded-lg border bg-white px-4 py-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 transition focus:outline-none focus:ring-2";

/** 输入框边框/聚焦状态：invalid 传 true 显示红色错误态 */
export function fieldState(invalid: boolean): string {
  return invalid
    ? "border-red-300 focus:border-red-300 focus:ring-red-100"
    : "border-neutral-200 focus:border-indigo-500 focus:ring-indigo-100";
}
```

- [ ] **Step 3: 创建 PasswordInput.tsx**

```tsx
"use client";

import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";

interface PasswordInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  id: string;
}

export function PasswordInput({ id, className = "", ...props }: PasswordInputProps) {
  const [show, setShow] = useState(false);
  return (
    <div className="relative">
      <input
        id={id}
        type={show ? "text" : "password"}
        className={`w-full rounded-lg border bg-white py-2.5 pl-4 pr-11 text-sm text-neutral-900 placeholder:text-neutral-400 transition focus:outline-none focus:ring-2 ${className}`}
        {...props}
      />
      <button
        type="button"
        onClick={() => setShow((s) => !s)}
        className="absolute inset-y-0 right-0 flex items-center pr-3 text-neutral-400 transition hover:text-neutral-600"
        tabIndex={-1}
        aria-label={show ? "隐藏密码" : "显示密码"}
      >
        {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  );
}
```

- [ ] **Step 4: 验证编译**

Run: `cd frontend && npm run build`
Expected: 构建成功（组件尚未被引用，此时应先跑 `npm run lint` 确认无未使用告警；如 lint 报未引用组件，可暂不处理，Task 3/4 引用后即消除）。

- [ ] **Step 5: Commit**

```bash
cd /home/chun/dev/projects/kada
git add frontend/src/components/auth/
git commit -m "feat: shared auth components (AuthCard/FormField/PasswordInput)"
```

---

### Task 3: 登录页重构 + 校验

**Files:**
- Modify: `frontend/src/app/(auth)/login/page.tsx`（整文件替换）

**Interfaces:**
- 消费：`AuthCard`、`FormField`、`inputBase`、`fieldState`、`PasswordInput`；`authAPI`、`setToken`、`setUser`（来自 `src/lib/api.ts` / `src/lib/auth.ts`，签名不变）

- [ ] **Step 1: 重写登录页**

将 `frontend/src/app/(auth)/login/page.tsx` 替换为：

```tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { authAPI } from "@/lib/api";
import { setToken, setUser } from "@/lib/auth";
import { AuthCard } from "@/components/auth/AuthCard";
import { FormField, inputBase, fieldState } from "@/components/auth/FormField";
import { PasswordInput } from "@/components/auth/PasswordInput";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function LoginPage() {
  const router = useRouter();
  const [tab, setTab] = useState<"phone" | "email">("phone");
  const [loading, setLoading] = useState(false);

  // 手机号登录
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [phoneError, setPhoneError] = useState("");
  const [codeError, setCodeError] = useState("");

  // 邮箱登录
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState("");
  const [passwordError, setPasswordError] = useState("");

  const phoneValid = phone.length === 11;

  const handleSendCode = async () => {
    if (phone.length !== 11) {
      setPhoneError("请输入 11 位手机号");
      return;
    }
    setPhoneError("");
    try {
      await authAPI.sendSMSCode(phone);
      setCodeSent(true);
      toast.success("验证码已发送");
      setCountdown(60);
      const timer = setInterval(() => {
        setCountdown((c) => {
          if (c <= 1) {
            clearInterval(timer);
            return 0;
          }
          return c - 1;
        });
      }, 1000);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "发送失败");
    }
  };

  const handlePhoneLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (phone.length !== 11) {
      setPhoneError("请输入 11 位手机号");
      ok = false;
    } else {
      setPhoneError("");
    }
    if (code.length !== 6) {
      setCodeError("请输入 6 位验证码");
      ok = false;
    } else {
      setCodeError("");
    }
    if (!ok) return;

    setLoading(true);
    try {
      const data = await authAPI.loginByPhone(phone, code);
      setToken(data.token);
      setUser(data.user);
      toast.success("登录成功");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "登录失败");
    } finally {
      setLoading(false);
    }
  };

  const handleEmailLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (!EMAIL_RE.test(email)) {
      setEmailError("请输入有效的邮箱地址");
      ok = false;
    } else {
      setEmailError("");
    }
    if (!password) {
      setPasswordError("请输入密码");
      ok = false;
    } else {
      setPasswordError("");
    }
    if (!ok) return;

    setLoading(true);
    try {
      const data = await authAPI.loginByEmail(email, password);
      setToken(data.token);
      setUser(data.user);
      toast.success("登录成功");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "登录失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthCard
      title="登录 Kada"
      subtitle="智能短链接管理平台"
      footer={
        <>
          还没有账号？{" "}
          <Link href="/register" className="font-medium text-indigo-600 hover:text-indigo-500">
            立即注册
          </Link>
        </>
      }
    >
      {/* 手机号 / 邮箱 分段切换 */}
      <div className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-neutral-100 p-1">
        <button
          type="button"
          onClick={() => setTab("phone")}
          className={`rounded-md py-2 text-sm font-medium transition ${
            tab === "phone"
              ? "bg-white text-neutral-900 shadow-sm"
              : "text-neutral-500 hover:text-neutral-700"
          }`}
        >
          手机号登录
        </button>
        <button
          type="button"
          onClick={() => setTab("email")}
          className={`rounded-md py-2 text-sm font-medium transition ${
            tab === "email"
              ? "bg-white text-neutral-900 shadow-sm"
              : "text-neutral-500 hover:text-neutral-700"
          }`}
        >
          邮箱登录
        </button>
      </div>

      {tab === "phone" ? (
        <form onSubmit={handlePhoneLogin} className="space-y-4" noValidate>
          <FormField id="phone" label="手机号" error={phoneError}>
            <input
              id="phone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
              className={`${inputBase} ${fieldState(!!phoneError)}`}
              placeholder="请输入 11 位手机号"
              required
            />
          </FormField>
          <FormField id="code" label="验证码" error={codeError}>
            <div className="flex gap-3">
              <input
                id="code"
                type="text"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                className={`${inputBase} ${fieldState(!!codeError)}`}
                placeholder="6 位验证码"
                required
              />
              <button
                type="button"
                onClick={handleSendCode}
                disabled={countdown > 0 || !phoneValid}
                className="shrink-0 rounded-lg bg-indigo-50 px-4 py-2.5 text-sm font-medium text-indigo-600 transition hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {countdown > 0 ? `${countdown}s` : codeSent ? "重新发送" : "获取验证码"}
              </button>
            </div>
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "登录中..." : "登录"}
          </button>
        </form>
      ) : (
        <form onSubmit={handleEmailLogin} className="space-y-4" noValidate>
          <FormField id="email" label="邮箱" error={emailError}>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={`${inputBase} ${fieldState(!!emailError)}`}
              placeholder="your@email.com"
              required
            />
          </FormField>
          <FormField id="password" label="密码" error={passwordError}>
            <PasswordInput
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={fieldState(!!passwordError)}
              placeholder="请输入密码"
              required
            />
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "登录中..." : "登录"}
          </button>
        </form>
      )}
    </AuthCard>
  );
}
```

- [ ] **Step 2: 验证编译与静态检查**

Run: `cd frontend && npm run lint && npm run build`
Expected: lint 无报错；build 成功。`/login` 手动冒烟：分段切换、手机号 11 位校验、验证码 6 位校验、密码显隐、错误内联提示。

- [ ] **Step 3: Commit**

```bash
cd /home/chun/dev/projects/kada
git add "frontend/src/app/(auth)/login/page.tsx"
git commit -m "feat: restyle login page with validation and password visibility"
```

---

### Task 4: 注册页重构 + 校验

**Files:**
- Modify: `frontend/src/app/(auth)/register/page.tsx`（整文件替换）

**Interfaces:**
- 消费：`AuthCard`、`FormField`、`inputBase`、`fieldState`、`PasswordInput`；`authAPI`、`setToken`、`setUser`

- [ ] **Step 1: 重写注册页**

将 `frontend/src/app/(auth)/register/page.tsx` 替换为：

```tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { authAPI } from "@/lib/api";
import { setToken, setUser } from "@/lib/auth";
import { AuthCard } from "@/components/auth/AuthCard";
import { FormField, inputBase, fieldState } from "@/components/auth/FormField";
import { PasswordInput } from "@/components/auth/PasswordInput";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export default function RegisterPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<{
    name?: string;
    email?: string;
    password?: string;
    confirm?: string;
  }>({});

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const errs: typeof errors = {};
    if (!name.trim()) errs.name = "请输入昵称";
    if (!EMAIL_RE.test(email)) errs.email = "请输入有效的邮箱地址";
    if (password.length < 6) errs.password = "密码至少 6 位";
    if (confirm !== password) errs.confirm = "两次输入的密码不一致";
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setLoading(true);
    try {
      const data = await authAPI.registerByEmail(email, password, name.trim());
      setToken(data.token);
      setUser(data.user);
      toast.success("注册成功！");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "注册失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthCard
      title="注册 Kada"
      subtitle="创建你的短链接管理账号"
      footer={
        <>
          已有账号？{" "}
          <Link href="/login" className="font-medium text-indigo-600 hover:text-indigo-500">
            立即登录
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <FormField id="name" label="昵称" error={errors.name}>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={`${inputBase} ${fieldState(!!errors.name)}`}
            placeholder="你的昵称"
            required
          />
        </FormField>
        <FormField id="email" label="邮箱" error={errors.email}>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={`${inputBase} ${fieldState(!!errors.email)}`}
            placeholder="your@email.com"
            required
          />
        </FormField>
        <FormField id="password" label="密码" error={errors.password}>
          <PasswordInput
            id="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className={fieldState(!!errors.password)}
            placeholder="至少 6 位"
            required
            minLength={6}
          />
        </FormField>
        <FormField id="confirm" label="确认密码" error={errors.confirm}>
          <PasswordInput
            id="confirm"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            className={fieldState(!!errors.confirm)}
            placeholder="再次输入密码"
            required
          />
        </FormField>
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {loading ? "注册中..." : "注册"}
        </button>
      </form>
    </AuthCard>
  );
}
```

- [ ] **Step 2: 验证编译与静态检查**

Run: `cd frontend && npm run lint && npm run build`
Expected: lint 无报错；build 成功。`/register` 手动冒烟：空昵称、邮箱格式、密码 <6 位、两次密码不一致均有内联红字；密码显隐正常。

- [ ] **Step 3: Commit**

```bash
cd /home/chun/dev/projects/kada
git add "frontend/src/app/(auth)/register/page.tsx"
git commit -m "feat: restyle register page with confirm password and validation"
```

---

### Task 5: 主页面重排

**Files:**
- Modify: `frontend/src/app/page.tsx`（整文件替换）

**Interfaces:**
- 无新增接口；仅重构现有 server component 的 JSX/样式

- [ ] **Step 1: 重写主页面**

将 `frontend/src/app/page.tsx` 替换为：

```tsx
import Link from "next/link";
import { Link2, Shield, Smartphone, Zap, BarChart3, Globe } from "lucide-react";

const FEATURES = [
  { icon: Zap, title: "智能短链", desc: "一键缩短长链接，支持自定义短码" },
  { icon: Globe, title: "全平台兼容", desc: "自动识别微信、QQ、小红书等平台，引导在浏览器打开" },
  { icon: BarChart3, title: "点击追踪", desc: "记录每次点击，追踪来源平台、设备和地理位置" },
  { icon: Smartphone, title: "短信友好", desc: "支持手机号验证码登录，短信链接不被拦截" },
  { icon: Shield, title: "安全可靠", desc: "HTTPS 加密传输，JWT 认证，数据隔离保障链接安全" },
  { icon: Link2, title: "API 开放", desc: "提供 RESTful API，方便集成到你的应用和工作流" },
];

export default function HomePage() {
  return (
    <div className="min-h-screen bg-white">
      {/* Nav */}
      <header className="sticky top-0 z-50 border-b border-neutral-200 bg-white">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-2">
            <div className="flex size-8 items-center justify-center rounded-lg bg-indigo-600">
              <Link2 className="size-4 text-white" />
            </div>
            <span className="text-lg font-semibold text-neutral-900">Kada</span>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/login"
              className="rounded-lg px-4 py-2 text-sm font-medium text-neutral-600 transition hover:bg-neutral-50 hover:text-neutral-900"
            >
              登录
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-700"
            >
              免费注册
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="mx-auto max-w-6xl px-4 py-24 sm:px-6 sm:py-32">
        <div className="mx-auto max-w-3xl text-center">
          <h1 className="text-4xl font-bold tracking-tight text-neutral-900 sm:text-5xl">
            你的链接<span className="text-indigo-600">，无处不在</span>
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-neutral-500">
            智能短链接管理平台，缩短、分享并追踪你的每一个链接。
            完美兼容微信、QQ、小红书等国内主流平台。
          </p>
          <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <Link
              href="/register"
              className="w-full rounded-lg bg-indigo-600 px-8 py-3 text-base font-medium text-white transition hover:bg-indigo-700 sm:w-auto"
            >
              免费开始使用
            </Link>
            <Link
              href="/login"
              className="w-full rounded-lg border border-neutral-200 bg-white px-8 py-3 text-base font-medium text-neutral-700 transition hover:border-neutral-300 hover:bg-neutral-50 sm:w-auto"
            >
              登录
            </Link>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="border-t border-neutral-100 bg-neutral-50">
        <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-24">
          <div className="mb-14 text-center">
            <h2 className="text-3xl font-bold tracking-tight text-neutral-900">
              一个平台，搞定所有链接
            </h2>
            <p className="mt-3 text-base text-neutral-500">为国内环境量身打造的短链接解决方案</p>
          </div>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map(({ icon: Icon, title, desc }) => (
              <div
                key={title}
                className="rounded-xl border border-neutral-200 bg-white p-6 transition-colors hover:border-neutral-300"
              >
                <div className="mb-4 flex size-10 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600">
                  <Icon className="size-5" />
                </div>
                <h3 className="mb-1.5 text-base font-semibold text-neutral-900">{title}</h3>
                <p className="text-sm leading-relaxed text-neutral-500">{desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-indigo-600">
        <div className="mx-auto max-w-6xl px-4 py-16 text-center sm:px-6">
          <h2 className="text-2xl font-bold text-white sm:text-3xl">准备好管理你的链接了吗？</h2>
          <p className="mt-3 text-base text-indigo-100">免费注册，几秒钟内创建你的第一个短链接</p>
          <Link
            href="/register"
            className="mt-8 inline-flex rounded-lg bg-white px-8 py-3 text-base font-medium text-indigo-600 transition hover:bg-indigo-50"
          >
            立即开始
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-neutral-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 px-4 py-10 sm:flex-row sm:px-6">
          <div className="flex items-center gap-2">
            <div className="flex size-6 items-center justify-center rounded bg-indigo-600">
              <Link2 className="size-3.5 text-white" />
            </div>
            <span className="text-sm font-medium text-neutral-700">Kada</span>
          </div>
          <p className="text-xs text-neutral-400">
            &copy; {new Date().getFullYear()} Kada. 保留所有权利。
          </p>
        </div>
      </footer>
    </div>
  );
}
```

- [ ] **Step 2: 验证编译与静态检查**

Run: `cd frontend && npm run lint && npm run build`
Expected: lint 无报错；build 成功。`/` 手动冒烟：无渐变、无徽章、特性卡片 hover 仅边框加深、CTA 为纯色块。

- [ ] **Step 3: Commit**

```bash
cd /home/chun/dev/projects/kada
git add frontend/src/app/page.tsx
git commit -m "feat: restyle landing page (clean minimal, remove gradients)"
```

---

### Task 6: 最终验证

**Files:**
- 无代码改动

- [ ] **Step 1: 全量 lint + build**

Run: `cd frontend && npm run lint && npm run build`
Expected: 全部通过，无警告。

- [ ] **Step 2: 运行时冒烟**

Run: `cd frontend && npm run dev`
手动检查：
- `/` 主页面：Header / Hero / 特性 / CTA / 页脚完整，样式统一克制
- `/login`：两个 tab 均可切换，验证码倒计时、密码显隐、内联错误、加载态正常；错误场景 toast 正常
- `/register`：四项校验（昵称/邮箱/密码/确认密码）内联提示；注册成功跳转 dashboard
- `/dashboard`：确认 AppLayout 未受影响

- [ ] **Step 3: 完成**

无额外 commit（各任务已独立提交）。
