# Frontend Homepage / Login & Register Pages Restyle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the three Kada pages — the homepage `/`, login `/login`, and register `/register` — into a unified, restrained minimalist style, and complete the login/register form validation and password show/hide features.

**Architecture:** First define semantic design tokens in `globals.css` using Tailwind v4 `@theme`; then add three shared components under `src/components/auth/` (`AuthCard`, `FormField`, `PasswordInput`); login/register reuse that shell, and the homepage is re-laid out with the same token set. Pure frontend changes, no backend API changes.

**Tech Stack:** Next.js 16.2.10 (App Router), React 19, Tailwind CSS v4, lucide-react.

**Testing approach note:** This project has no component-level test framework (no vitest/jest). The verification cycle for each task is `npm run lint` + `npm run build` (type checking + compilation), plus manual browser smoke testing.

## Global Constraints

- Never touch the backend API (the API names/signatures in `src/lib/api.ts` stay unchanged)
- Keep the indigo palette; only remove gradients (`bg-gradient-to-*`), remove `shadow-lg`/`hover:shadow-lg`, and remove `rounded-2xl`
- Borders uniformly `neutral-200` hairline; backgrounds uniformly `neutral-50` (page base) / `white` (cards and blocks)
- Two border-radius tiers: buttons/inputs `rounded-lg` (8px), cards `rounded-xl` (12px)
- Put new files in `src/components/auth/`
- First read the client-component/form-related guides in `node_modules/next/dist/docs/` to confirm this change is unaffected by Next 16 breaking changes (all pages in this project already use `"use client"`, so no route structure changes are needed)

---
## File Structure

- Modify: `frontend/src/app/globals.css` — design tokens
- Create: `frontend/src/components/auth/AuthCard.tsx`
- Create: `frontend/src/components/auth/FormField.tsx` (contains the `FormField` component + the `inputBase` / `fieldState` utilities)
- Create: `frontend/src/components/auth/PasswordInput.tsx`
- Modify: `frontend/src/app/page.tsx` — homepage re-layout
- Modify: `frontend/src/app/(auth)/login/page.tsx` — login page refactor + validation
- Modify: `frontend/src/app/(auth)/register/page.tsx` — register page refactor + validation

---

### Task 1: Design Tokens (globals.css)

**Files:**
- Modify: `frontend/src/app/globals.css`

**Interfaces:**
- Produces: Tailwind semantic utility classes `bg-brand` / `text-brand` / `border-hairline` / `bg-surface` (this task only declares them; later tasks use them as needed; you may also use the standard indigo/neutral classes directly — the two are equivalent)

- [ ] **Step 1: Rewrite globals.css**

Replace `frontend/src/app/globals.css` with:

```css
@import "tailwindcss";

@theme {
  --color-brand: #4f46e5;        /* indigo-600 primary color */
  --color-brand-dark: #4338ca;   /* indigo-700 */
  --color-brand-soft: #eef2ff;   /* indigo-50 */
  --color-hairline: #e5e5e5;     /* neutral-200 hairline border */
  --color-surface: #fafafa;      /* neutral-50 page base */
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

- [ ] **Step 2: Verify compilation**

Run: `cd frontend && npm run build`
Expected: build succeeds (no page changes in this task; only confirms the token definitions do not break Tailwind).

- [ ] **Step 3: Commit**

```bash
cd /home/chun/dev/projects/kada
git add frontend/src/app/globals.css
git commit -m "style: add semantic design tokens (brand/hairline/surface)"
```

---

### Task 2: Shared Auth Components

**Files:**
- Create: `frontend/src/components/auth/AuthCard.tsx`
- Create: `frontend/src/components/auth/FormField.tsx`
- Create: `frontend/src/components/auth/PasswordInput.tsx`

**Interfaces:**
- Produces:
  - `AuthCard({ title, subtitle, footer, children })` — white card shell
  - `FormField({ id, label, error, children })` — label + input + inline error
  - `inputBase` — base input class string
  - `fieldState(invalid: boolean)` — input border/focus state class string
  - `PasswordInput({ id, className, ...inputProps })` — password field + show/hide toggle

- [ ] **Step 1: Create AuthCard.tsx**

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

- [ ] **Step 2: Create FormField.tsx**

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

/** Base input styles (excluding border color / focus state) */
export const inputBase =
  "w-full rounded-lg border bg-white px-4 py-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 transition focus:outline-none focus:ring-2";

/** Input border/focus state: pass true for invalid to show the red error state */
export function fieldState(invalid: boolean): string {
  return invalid
    ? "border-red-300 focus:border-red-300 focus:ring-red-100"
    : "border-neutral-200 focus:border-indigo-500 focus:ring-indigo-100";
}
```

- [ ] **Step 3: Create PasswordInput.tsx**

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
        aria-label={show ? "Hide password" : "Show password"}
      >
        {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  );
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd frontend && npm run build`
Expected: build succeeds (the components are not referenced yet; at this point you should first run `npm run lint` to confirm there are no unused warnings; if lint reports unreferenced components, you may leave it for now — it disappears once Task 3/4 reference them).

- [ ] **Step 5: Commit**

```bash
cd /home/chun/dev/projects/kada
git add frontend/src/components/auth/
git commit -m "feat: shared auth components (AuthCard/FormField/PasswordInput)"
```

---

### Task 3: Login Page Refactor + Validation

**Files:**
- Modify: `frontend/src/app/(auth)/login/page.tsx` (full-file replacement)

**Interfaces:**
- Consumes: `AuthCard`, `FormField`, `inputBase`, `fieldState`, `PasswordInput`; `authAPI`, `setToken`, `setUser` (from `src/lib/api.ts` / `src/lib/auth.ts`, signatures unchanged)

- [ ] **Step 1: Rewrite the login page**

Replace `frontend/src/app/(auth)/login/page.tsx` with:

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

  // Phone login
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [phoneError, setPhoneError] = useState("");
  const [codeError, setCodeError] = useState("");

  // Email login
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState("");
  const [passwordError, setPasswordError] = useState("");

  const phoneValid = phone.length === 11;

  const handleSendCode = async () => {
    if (phone.length !== 11) {
      setPhoneError("Please enter an 11-digit phone number");
      return;
    }
    setPhoneError("");
    try {
      await authAPI.sendSMSCode(phone);
      setCodeSent(true);
      toast.success("Verification code sent");
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
      toast.error(e instanceof Error ? e.message : "Failed to send");
    }
  };

  const handlePhoneLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (phone.length !== 11) {
      setPhoneError("Please enter an 11-digit phone number");
      ok = false;
    } else {
      setPhoneError("");
    }
    if (code.length !== 6) {
      setCodeError("Please enter the 6-digit verification code");
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
      toast.success("Logged in successfully");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  const handleEmailLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    let ok = true;
    if (!EMAIL_RE.test(email)) {
      setEmailError("Please enter a valid email address");
      ok = false;
    } else {
      setEmailError("");
    }
    if (!password) {
      setPasswordError("Please enter your password");
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
      toast.success("Logged in successfully");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthCard
      title="Log in to Kada"
      subtitle="Smart short link management platform"
      footer={
        <>
          Don&apos;t have an account?{" "}
          <Link href="/register" className="font-medium text-indigo-600 hover:text-indigo-500">
            Sign up now
          </Link>
        </>
      }
    >
      {/* Phone / email segmented toggle */}
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
          Phone login
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
          Email login
        </button>
      </div>

      {tab === "phone" ? (
        <form onSubmit={handlePhoneLogin} className="space-y-4" noValidate>
          <FormField id="phone" label="Phone number" error={phoneError}>
            <input
              id="phone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
              className={`${inputBase} ${fieldState(!!phoneError)}`}
              placeholder="Enter an 11-digit phone number"
              required
            />
          </FormField>
          <FormField id="code" label="Verification code" error={codeError}>
            <div className="flex gap-3">
              <input
                id="code"
                type="text"
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                className={`${inputBase} ${fieldState(!!codeError)}`}
                placeholder="6-digit code"
                required
              />
              <button
                type="button"
                onClick={handleSendCode}
                disabled={countdown > 0 || !phoneValid}
                className="shrink-0 rounded-lg bg-indigo-50 px-4 py-2.5 text-sm font-medium text-indigo-600 transition hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {countdown > 0 ? `${countdown}s` : codeSent ? "Resend" : "Get code"}
              </button>
            </div>
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "Logging in..." : "Log in"}
          </button>
        </form>
      ) : (
        <form onSubmit={handleEmailLogin} className="space-y-4" noValidate>
          <FormField id="email" label="Email" error={emailError}>
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
          <FormField id="password" label="Password" error={passwordError}>
            <PasswordInput
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={fieldState(!!passwordError)}
              placeholder="Enter your password"
              required
            />
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "Logging in..." : "Log in"}
          </button>
        </form>
      )}
    </AuthCard>
  );
}
```

- [ ] **Step 2: Verify compilation and static checks**

Run: `cd frontend && npm run lint && npm run build`
Expected: lint reports no errors; build succeeds. Manual smoke test of `/login`: segmented toggle, 11-digit phone validation, 6-digit code validation, password show/hide, inline error messages.

- [ ] **Step 3: Commit**

```bash
cd /home/chun/dev/projects/kada
git add "frontend/src/app/(auth)/login/page.tsx"
git commit -m "feat: restyle login page with validation and password visibility"
```

---

### Task 4: Register Page Refactor + Validation

**Files:**
- Modify: `frontend/src/app/(auth)/register/page.tsx` (full-file replacement)

**Interfaces:**
- Consumes: `AuthCard`, `FormField`, `inputBase`, `fieldState`, `PasswordInput`; `authAPI`, `setToken`, `setUser`

- [ ] **Step 1: Rewrite the register page**

Replace `frontend/src/app/(auth)/register/page.tsx` with:

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
    if (!name.trim()) errs.name = "Please enter a nickname";
    if (!EMAIL_RE.test(email)) errs.email = "Please enter a valid email address";
    if (password.length < 6) errs.password = "Password must be at least 6 characters";
    if (confirm !== password) errs.confirm = "The two passwords do not match";
    setErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setLoading(true);
    try {
      const data = await authAPI.registerByEmail(email, password, name.trim());
      setToken(data.token);
      setUser(data.user);
      toast.success("Registration successful!");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthCard
      title="Sign up for Kada"
      subtitle="Create your short link management account"
      footer={
        <>
          Already have an account?{" "}
          <Link href="/login" className="font-medium text-indigo-600 hover:text-indigo-500">
            Log in now
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        <FormField id="name" label="Nickname" error={errors.name}>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className={`${inputBase} ${fieldState(!!errors.name)}`}
            placeholder="Your nickname"
            required
          />
        </FormField>
        <FormField id="email" label="Email" error={errors.email}>
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
        <FormField id="password" label="Password" error={errors.password}>
          <PasswordInput
            id="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className={fieldState(!!errors.password)}
            placeholder="At least 6 characters"
            required
            minLength={6}
          />
        </FormField>
        <FormField id="confirm" label="Confirm password" error={errors.confirm}>
          <PasswordInput
            id="confirm"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            className={fieldState(!!errors.confirm)}
            placeholder="Enter the password again"
            required
          />
        </FormField>
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {loading ? "Signing up..." : "Sign up"}
        </button>
      </form>
    </AuthCard>
  );
}
```

- [ ] **Step 2: Verify compilation and static checks**

Run: `cd frontend && npm run lint && npm run build`
Expected: lint reports no errors; build succeeds. Manual smoke test of `/register`: empty nickname, email format, password < 6 characters, and mismatched passwords all show inline red text; password show/hide works.

- [ ] **Step 3: Commit**

```bash
cd /home/chun/dev/projects/kada
git add "frontend/src/app/(auth)/register/page.tsx"
git commit -m "feat: restyle register page with confirm password and validation"
```

---

### Task 5: Homepage Re-layout

**Files:**
- Modify: `frontend/src/app/page.tsx` (full-file replacement)

**Interfaces:**
- No new interfaces; only refactors the JSX/styles of the existing server component

- [ ] **Step 1: Rewrite the homepage**

Replace `frontend/src/app/page.tsx` with:

```tsx
import Link from "next/link";
import { Link2, Shield, Smartphone, Zap, BarChart3, Globe } from "lucide-react";

const FEATURES = [
  { icon: Zap, title: "Smart short links", desc: "Shorten long links in one click, with custom short codes" },
  { icon: Globe, title: "Works on every platform", desc: "Automatically detects WeChat, QQ, Xiaohongshu and other platforms and prompts opening in a browser" },
  { icon: BarChart3, title: "Click tracking", desc: "Records every click and tracks the referring platform, device, and geographic location" },
  { icon: Smartphone, title: "SMS friendly", desc: "Supports phone-number verification code login, so SMS links are not blocked" },
  { icon: Shield, title: "Safe and reliable", desc: "HTTPS encrypted transport, JWT authentication, and data isolation keep links secure" },
  { icon: Link2, title: "Open API", desc: "Provides a RESTful API that is easy to integrate into your apps and workflows" },
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
              Log in
            </Link>
            <Link
              href="/register"
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-indigo-700"
            >
              Sign up free
            </Link>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="mx-auto max-w-6xl px-4 py-24 sm:px-6 sm:py-32">
        <div className="mx-auto max-w-3xl text-center">
          <h1 className="text-4xl font-bold tracking-tight text-neutral-900 sm:text-5xl">
            Your links<span className="text-indigo-600">, everywhere</span>
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-lg leading-relaxed text-neutral-500">
            A smart short link management platform that shortens, shares, and tracks every one of your links.
            Fully compatible with mainstream Chinese platforms such as WeChat, QQ, and Xiaohongshu.
          </p>
          <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <Link
              href="/register"
              className="w-full rounded-lg bg-indigo-600 px-8 py-3 text-base font-medium text-white transition hover:bg-indigo-700 sm:w-auto"
            >
              Start for free
            </Link>
            <Link
              href="/login"
              className="w-full rounded-lg border border-neutral-200 bg-white px-8 py-3 text-base font-medium text-neutral-700 transition hover:border-neutral-300 hover:bg-neutral-50 sm:w-auto"
            >
              Log in
            </Link>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="border-t border-neutral-100 bg-neutral-50">
        <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-24">
          <div className="mb-14 text-center">
            <h2 className="text-3xl font-bold tracking-tight text-neutral-900">
              One platform for all your links
            </h2>
            <p className="mt-3 text-base text-neutral-500">A short link solution built specifically for the Chinese internet environment</p>
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
          <h2 className="text-2xl font-bold text-white sm:text-3xl">Ready to manage your links?</h2>
          <p className="mt-3 text-base text-indigo-100">Sign up for free and create your first short link in seconds</p>
          <Link
            href="/register"
            className="mt-8 inline-flex rounded-lg bg-white px-8 py-3 text-base font-medium text-indigo-600 transition hover:bg-indigo-50"
          >
            Get started
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
            &copy; {new Date().getFullYear()} Kada. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}
```

- [ ] **Step 2: Verify compilation and static checks**

Run: `cd frontend && npm run lint && npm run build`
Expected: lint reports no errors; build succeeds. Manual smoke test of `/`: no gradients, no badges, feature cards darken the border only on hover, CTA is a solid color block.

- [ ] **Step 3: Commit**

```bash
cd /home/chun/dev/projects/kada
git add frontend/src/app/page.tsx
git commit -m "feat: restyle landing page (clean minimal, remove gradients)"
```

---

### Task 6: Final Verification

**Files:**
- No code changes

- [ ] **Step 1: Full lint + build**

Run: `cd frontend && npm run lint && npm run build`
Expected: everything passes with no warnings.

- [ ] **Step 2: Runtime smoke test**

Run: `cd frontend && npm run dev`
Check manually:
- `/` homepage: Header / Hero / Features / CTA / Footer all complete, styling unified and restrained
- `/login`: both tabs switch, verification code countdown, password show/hide, inline errors, and loading state all work; the toast works in error scenarios
- `/register`: inline messages for all four validations (nickname/email/password/confirm password); successful registration redirects to the dashboard
- `/dashboard`: confirm AppLayout is unaffected

- [ ] **Step 3: Done**

No additional commit (each task was already committed independently).
