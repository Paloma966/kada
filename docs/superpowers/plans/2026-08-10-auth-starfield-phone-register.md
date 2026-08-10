# 登录/注册页星空背景 + 手机号注册 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让登录/注册页套用首页的 3D 星空背景（透明玻璃卡片），并给注册页增加手机号注册入口（手机号+验证码+昵称）。

**Architecture:** 复用首页的 `StarfieldCanvas` + `bg-deep-space`。AuthCard 是登录/注册共用的外壳，改成星空背景 + 透明玻璃卡片即可两页同时生效；表单组件（FormField/PasswordInput）同步改暗色。手机号注册复用现有后端 `login-by-phone`（自动注册）+ `PATCH /me`（写昵称），**后端零改动**。

**Tech Stack:** Next.js 16, React 19, Tailwind v4, Three.js（已装），lucide-react。

**Spec:** `docs/superpowers/specs/2026-08-10-auth-starfield-phone-register-design.md`

## Global Constraints

- 定制版 Next.js（见 `frontend/AGENTS.md`）：写 React/Next 代码前先读 `frontend/node_modules/next/dist/docs/01-app/03-api-reference/01-directives/use-client.md`。
- 客户端组件顶部声明 `"use client"`（auth 页均为客户端组件）。
- 验证：`npm run build` MUST pass；`npm run test`（现有 18 项）必须仍通过。**不要**全仓 `npm run lint`（`src/app/dashboard/*` 有既有失败，非本功能引入）。
- 不引入外部字体；深空主题 `bg-deep-space`；品牌 indigo `#4f46e5`。
- 后端零改动（回归确认：`cd backend && go test ./...` 可选）。
- AuthCard/FormField/PasswordInput 仅被 login/register 使用（已验证），可安全改为暗色。
- 透明玻璃卡片样式：`rounded-xl border border-white/10 bg-white/[0.04] p-6 backdrop-blur-lg sm:p-8`。

---

### Task 1: 认证组件改暗色 + 星空背景（AuthCard / FormField / PasswordInput）

**Files:**
- Modify: `frontend/src/components/auth/AuthCard.tsx`
- Modify: `frontend/src/components/auth/FormField.tsx`
- Modify: `frontend/src/components/auth/PasswordInput.tsx`

**Interfaces:**
- Consumes: `StarfieldCanvas`（`@/components/StarfieldCanvas`，默认导出，className prop）
- Produces: `AuthCard` / `FormField` / `PasswordInput` 保持导出签名不变（login/register 页无需改 import）；`inputBase` / `fieldState` 导出签名不变

- [ ] **Step 1: 先读定制版 Next.js 的 use-client 文档**

```bash
sed -n '1,80p' frontend/node_modules/next/dist/docs/01-app/03-api-reference/01-directives/use-client.md
```

- [ ] **Step 2: 重写 AuthCard.tsx**

Replace `frontend/src/components/auth/AuthCard.tsx` entirely:

```tsx
import { Link2 } from "lucide-react";
import StarfieldCanvas from "@/components/StarfieldCanvas";

interface AuthCardProps {
  title: string;
  subtitle: string;
  footer: React.ReactNode;
  children: React.ReactNode;
}

export function AuthCard({ title, subtitle, footer, children }: AuthCardProps) {
  return (
    <div className="relative min-h-dvh overflow-x-hidden bg-deep-space">
      {/* 星空背景固定铺满，滚动时背景不动 */}
      <StarfieldCanvas className="fixed inset-0 h-full w-full" />
      <div className="relative z-10 flex min-h-dvh items-center justify-center px-4 py-12">
        <div className="w-full max-w-md">
          <div className="mb-8 flex flex-col items-center text-center">
            <div className="mb-4 flex size-11 items-center justify-center rounded-xl bg-indigo-600">
              <Link2 className="size-5 text-white" />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight text-white">{title}</h1>
            <p className="mt-2 text-sm text-indigo-200/80">{subtitle}</p>
          </div>
          {/* 透明玻璃卡片 */}
          <div className="rounded-xl border border-white/10 bg-white/[0.04] p-6 backdrop-blur-lg sm:p-8">
            {children}
          </div>
          <p className="mt-6 text-center text-sm text-indigo-100/70">{footer}</p>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: 重写 FormField.tsx**

Replace `frontend/src/components/auth/FormField.tsx` entirely:

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
      <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-neutral-100">
        {label}
      </label>
      {children}
      {error && <p className="mt-1.5 text-xs text-red-400">{error}</p>}
    </div>
  );
}

/** 输入框基础样式（不含边框色/聚焦态）——暗色星空主题 */
export const inputBase =
  "w-full rounded-lg border bg-white/10 px-4 py-2.5 text-sm text-white placeholder:text-neutral-400/70 transition focus:outline-none focus:ring-2";

/** 输入框边框/聚焦状态：invalid 传 true 显示红色错误态 */
export function fieldState(invalid: boolean): string {
  return invalid
    ? "border-red-400/60 focus:border-red-400 focus:ring-red-400/30"
    : "border-white/15 focus:border-indigo-400 focus:ring-indigo-400/30";
}
```

- [ ] **Step 4: 重写 PasswordInput.tsx**

Replace `frontend/src/components/auth/PasswordInput.tsx` entirely:

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
        className={`w-full rounded-lg border bg-white/10 py-2.5 pl-4 pr-11 text-sm text-white placeholder:text-neutral-400/70 transition focus:outline-none focus:ring-2 ${className}`}
        {...props}
      />
      <button
        type="button"
        onClick={() => setShow((s) => !s)}
        className="absolute inset-y-0 right-0 flex items-center pr-3 text-neutral-400 transition hover:text-neutral-200"
        tabIndex={-1}
        aria-label={show ? "隐藏密码" : "显示密码"}
      >
        {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  );
}
```

- [ ] **Step 5: lint（仅改动文件）+ 构建验证**

Run: `cd frontend && npx eslint src/components/auth/AuthCard.tsx src/components/auth/FormField.tsx src/components/auth/PasswordInput.tsx && npm run build`

Expected: eslint exit 0；build 通过。

- [ ] **Step 6: 提交**

```bash
git add src/components/auth/AuthCard.tsx src/components/auth/FormField.tsx src/components/auth/PasswordInput.tsx
git commit -m "feat: dark starfield theme for auth components (glass card)"
```

---

### Task 2: 登录页暗色样式微调

**Files:**
- Modify: `frontend/src/app/(auth)/login/page.tsx`

**Interfaces:**
- Consumes: `inputBase`/`fieldState`（Task 1 已暗色化，无需改动 import 或逻辑）

- [ ] **Step 1: 四处在原有文件内精确替换**

In `frontend/src/app/(auth)/login/page.tsx`:

(1) Footer 链接颜色（暗底上用浅 indigo）：

```diff
-          <Link href="/register" className="font-medium text-indigo-600 hover:text-indigo-500">
+          <Link href="/register" className="font-medium text-indigo-300 hover:text-indigo-200">
```

(2) 分段 tab 容器底 + 未选中态：

```diff
-      <div className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-neutral-100 p-1">
+      <div className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-white/10 p-1">
```

```diff
-              : "text-neutral-500 hover:text-neutral-700"
+              : "text-neutral-300 hover:text-white"
```

(3) 「获取验证码」按钮：

```diff
-                className="shrink-0 rounded-lg bg-indigo-50 px-4 py-2.5 text-sm font-medium text-indigo-600 transition hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-50"
+                className="shrink-0 rounded-lg bg-indigo-500/20 px-4 py-2.5 text-sm font-medium text-indigo-200 transition hover:bg-indigo-500/30 disabled:cursor-not-allowed disabled:opacity-50"
```

(4) 确认两处 tab 按钮的「选中态」仍为 `bg-white text-neutral-900 shadow-sm`（保持白色选中块，无需改）。

- [ ] **Step 2: 构建验证**

Run: `cd frontend && npm run build`

Expected: build 通过。

- [ ] **Step 3: 提交**

```bash
git add "src/app/(auth)/login/page.tsx"
git commit -m "style: dark theme tweaks on login page"
```

---

### Task 3: 注册页加手机号/邮箱 tab + 手机号注册

**Files:**
- Modify: `frontend/src/app/(auth)/register/page.tsx`

**Interfaces:**
- Consumes:
  - `authAPI.sendSMSCode(phone)`（`@/lib/api`）
  - `authAPI.loginByPhone(phone, code)` → `{ token, user }`（后端自动注册）
  - `authAPI.updateMe(token, { name })`
  - `setToken` / `setUser`（`@/lib/auth`）
  - `AuthCard` / `FormField` / `inputBase` / `fieldState` / `PasswordInput`
- Produces: 注册页支持手机号注册（默认手机号 tab）与邮箱注册

- [ ] **Step 1: 重写 register/page.tsx**

Replace `frontend/src/app/(auth)/register/page.tsx` entirely:

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
  const [tab, setTab] = useState<"phone" | "email">("phone");
  const [loading, setLoading] = useState(false);

  // 手机号注册
  const [phoneName, setPhoneName] = useState("");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [phoneNameError, setPhoneNameError] = useState("");
  const [phoneError, setPhoneError] = useState("");
  const [codeError, setCodeError] = useState("");

  // 邮箱注册
  const [emailName, setEmailName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [emailErrors, setEmailErrors] = useState<{
    name?: string;
    email?: string;
    password?: string;
    confirm?: string;
  }>({});

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

  const handlePhoneRegister = async (e: React.FormEvent) => {
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
      // 昵称：输入了就用输入的，否则默认「用户+手机号后4位」
      const nickname = phoneName.trim() || `用户${phone.slice(-4)}`;
      if (!data.user.name) {
        await authAPI.updateMe(data.token, { name: nickname });
      }
      setToken(data.token);
      setUser({ ...data.user, name: data.user.name || nickname });
      toast.success("注册成功！");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "注册失败");
    } finally {
      setLoading(false);
    }
  };

  const handleEmailRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    const errs: typeof emailErrors = {};
    if (!emailName.trim()) errs.name = "请输入昵称";
    if (!EMAIL_RE.test(email)) errs.email = "请输入有效的邮箱地址";
    if (password.length < 6) errs.password = "密码至少 6 位";
    if (confirm !== password) errs.confirm = "两次输入的密码不一致";
    setEmailErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setLoading(true);
    try {
      const data = await authAPI.registerByEmail(email, password, emailName.trim());
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
          <Link href="/login" className="font-medium text-indigo-300 hover:text-indigo-200">
            立即登录
          </Link>
        </>
      }
    >
      {/* 手机号 / 邮箱 分段切换 */}
      <div className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-white/10 p-1">
        <button
          type="button"
          onClick={() => setTab("phone")}
          className={`rounded-md py-2 text-sm font-medium transition ${
            tab === "phone"
              ? "bg-white text-neutral-900 shadow-sm"
              : "text-neutral-300 hover:text-white"
          }`}
        >
          手机号注册
        </button>
        <button
          type="button"
          onClick={() => setTab("email")}
          className={`rounded-md py-2 text-sm font-medium transition ${
            tab === "email"
              ? "bg-white text-neutral-900 shadow-sm"
              : "text-neutral-300 hover:text-white"
          }`}
        >
          邮箱注册
        </button>
      </div>

      {tab === "phone" ? (
        <form onSubmit={handlePhoneRegister} className="space-y-4" noValidate>
          <FormField id="pname" label="昵称" error={phoneNameError}>
            <input
              id="pname"
              type="text"
              value={phoneName}
              onChange={(e) => setPhoneName(e.target.value)}
              className={`${inputBase} ${fieldState(!!phoneNameError)}`}
              placeholder="你的昵称（可选）"
            />
          </FormField>
          <FormField id="pphone" label="手机号" error={phoneError}>
            <input
              id="pphone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
              className={`${inputBase} ${fieldState(!!phoneError)}`}
              placeholder="请输入 11 位手机号"
              required
            />
          </FormField>
          <FormField id="pcode" label="验证码" error={codeError}>
            <div className="flex gap-3">
              <input
                id="pcode"
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
                className="shrink-0 rounded-lg bg-indigo-500/20 px-4 py-2.5 text-sm font-medium text-indigo-200 transition hover:bg-indigo-500/30 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {countdown > 0 ? `${countdown}s` : codeSent ? "重新发送" : "获取验证码"}
              </button>
            </div>
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "注册中..." : "注册"}
          </button>
        </form>
      ) : (
        <form onSubmit={handleEmailRegister} className="space-y-4" noValidate>
          <FormField id="name" label="昵称" error={emailErrors.name}>
            <input
              id="name"
              type="text"
              value={emailName}
              onChange={(e) => setEmailName(e.target.value)}
              className={`${inputBase} ${fieldState(!!emailErrors.name)}`}
              placeholder="你的昵称"
              required
            />
          </FormField>
          <FormField id="email" label="邮箱" error={emailErrors.email}>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={`${inputBase} ${fieldState(!!emailErrors.email)}`}
              placeholder="your@email.com"
              required
            />
          </FormField>
          <FormField id="password" label="密码" error={emailErrors.password}>
            <PasswordInput
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={fieldState(!!emailErrors.password)}
              placeholder="至少 6 位"
              required
              minLength={6}
            />
          </FormField>
          <FormField id="confirm" label="确认密码" error={emailErrors.confirm}>
            <PasswordInput
              id="confirm"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              className={fieldState(!!emailErrors.confirm)}
              placeholder="再次输入密码"
              required
            />
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "注册中..." : "注册"}
          </button>
        </form>
      )}
    </AuthCard>
  );
}
```

- [ ] **Step 2: 构建验证**

Run: `cd frontend && npm run test && npm run build`

Expected: 18/18 测试通过；build 通过。

- [ ] **Step 3: 手动验证（dev 模式，需后端在跑）**

```bash
cd frontend && npm run dev
```

- [ ] 打开 `/register`：星空背景 + 透明玻璃卡片；默认手机号 tab；切换邮箱 tab 正常
- [ ] 手机号注册：输手机号 → 获取验证码（dev 后端日志可见 `📱 [DEV] ... Code:`）→ 填验证码+昵称 → 注册成功跳 `/dashboard`，侧边栏显示昵称
- [ ] 留空昵称注册：昵称自动为「用户+后4位」
- [ ] 已注册手机号再走注册：直接登录，不覆盖已有昵称
- [ ] `/login`：星空背景 + 玻璃卡片；手机号/邮箱登录均正常；密码显隐正常
- [ ] 小屏滚动时星空背景固定
- [ ] 控制台无报错

- [ ] **Step 4: 提交**

```bash
git add "src/app/(auth)/register/page.tsx"
git commit -m "feat: phone registration on register page (sms code + nickname)"
```

---

## 自检记录

- **Spec 覆盖**：星空背景（Task 1 AuthCard）、透明玻璃卡片（Task 1）、暗色表单组件（Task 1）、登录页暗色微调（Task 2）、手机号注册 tab+流程+默认昵称（Task 3）、后端零改动（计划约束）、错误处理 toast（各 Task）。
- **占位符扫描**：无 TBD/TODO；每个改动的完整代码均已给出。
- **类型一致性**：`authAPI.sendSMSCode` / `loginByPhone` / `updateMe` / `setToken` / `setUser` / `AuthCard` / `FormField` / `inputBase` / `fieldState` / `PasswordInput` 签名与现有代码一致；Task 3 消费 Task 1 的组件，无需改 import。
