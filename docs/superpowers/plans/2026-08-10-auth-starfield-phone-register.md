# Login/Register Pages Starfield Background + Phone Registration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the login/register pages adopt the homepage's 3D starfield background (transparent glass card), and add a phone registration entry point to the register page (phone number + verification code + nickname).

**Architecture:** Reuse the homepage's `StarfieldCanvas` + `bg-deep-space`. AuthCard is the shell shared by login/register, so changing it to a starfield background + transparent glass card makes both pages take effect at once; the form components (FormField/PasswordInput) are switched to dark at the same time. Phone registration reuses the existing backend `login-by-phone` (auto-registration) + `PATCH /me` (to write the nickname), with **zero backend changes**.

**Tech Stack:** Next.js 16, React 19, Tailwind v4, Three.js (already installed), lucide-react.

**Spec:** `docs/superpowers/specs/2026-08-10-auth-starfield-phone-register-design.md`

## Global Constraints

- Customized Next.js (see `frontend/AGENTS.md`): before writing React/Next code, read `frontend/node_modules/next/dist/docs/01-app/03-api-reference/01-directives/use-client.md`.
- Declare `"use client"` at the top of client components (the auth pages are all client components).
- Verification: `npm run build` MUST pass; `npm run test` (the existing 18 cases) must still pass. Do **not** run repo-wide `npm run lint` (`src/app/dashboard/*` has pre-existing failures, not introduced by this feature).
- Do not introduce external fonts; the deep space theme is `bg-deep-space`; the brand indigo is `#4f46e5`.
- Zero backend changes (regression check: `cd backend && go test ./...` optional).
- AuthCard/FormField/PasswordInput are used only by login/register (already verified), so they can safely be changed to dark.
- Transparent glass card style: `rounded-xl border border-white/10 bg-white/[0.04] p-6 backdrop-blur-lg sm:p-8`.

---

### Task 1: Switch Auth Components to Dark + Starfield Background (AuthCard / FormField / PasswordInput)

**Files:**
- Modify: `frontend/src/components/auth/AuthCard.tsx`
- Modify: `frontend/src/components/auth/FormField.tsx`
- Modify: `frontend/src/components/auth/PasswordInput.tsx`

**Interfaces:**
- Consumes: `StarfieldCanvas` (`@/components/StarfieldCanvas`, default export, className prop)
- Produces: `AuthCard` / `FormField` / `PasswordInput` keep their export signatures unchanged (the login/register pages need no import changes); `inputBase` / `fieldState` export signatures unchanged

- [ ] **Step 1: First read the customized Next.js use-client docs**

```bash
sed -n '1,80p' frontend/node_modules/next/dist/docs/01-app/03-api-reference/01-directives/use-client.md
```

- [ ] **Step 2: Rewrite AuthCard.tsx**

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
      {/* The starfield background is fixed and fills the screen, so it stays put while scrolling */}
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
          {/* Transparent glass card */}
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

- [ ] **Step 3: Rewrite FormField.tsx**

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

/** Base input styles (excluding border color / focus state) — dark starfield theme */
export const inputBase =
  "w-full rounded-lg border bg-white/10 px-4 py-2.5 text-sm text-white placeholder:text-neutral-400/70 transition focus:outline-none focus:ring-2";

/** Input border/focus state: pass true for invalid to show the red error state */
export function fieldState(invalid: boolean): string {
  return invalid
    ? "border-red-400/60 focus:border-red-400 focus:ring-red-400/30"
    : "border-white/15 focus:border-indigo-400 focus:ring-indigo-400/30";
}
```

- [ ] **Step 4: Rewrite PasswordInput.tsx**

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
        aria-label={show ? "Hide password" : "Show password"}
      >
        {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  );
}
```

- [ ] **Step 5: lint (changed files only) + build verification**

Run: `cd frontend && npx eslint src/components/auth/AuthCard.tsx src/components/auth/FormField.tsx src/components/auth/PasswordInput.tsx && npm run build`

Expected: eslint exits 0; build passes.

- [ ] **Step 6: Commit**

```bash
git add src/components/auth/AuthCard.tsx src/components/auth/FormField.tsx src/components/auth/PasswordInput.tsx
git commit -m "feat: dark starfield theme for auth components (glass card)"
```

---

### Task 2: Dark Style Tweaks on the Login Page

**Files:**
- Modify: `frontend/src/app/(auth)/login/page.tsx`

**Interfaces:**
- Consumes: `inputBase`/`fieldState` (already dark after Task 1; no import or logic changes needed)

- [ ] **Step 1: Four precise replacements within the existing file**

In `frontend/src/app/(auth)/login/page.tsx`:

(1) Footer link color (use a light indigo on the dark background):

```diff
-          <Link href="/register" className="font-medium text-indigo-600 hover:text-indigo-500">
+          <Link href="/register" className="font-medium text-indigo-300 hover:text-indigo-200">
```

(2) Segmented tab container background + unselected state:

```diff
-      <div className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-neutral-100 p-1">
+      <div className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-white/10 p-1">
```

```diff
-              : "text-neutral-500 hover:text-neutral-700"
+              : "text-neutral-300 hover:text-white"
```

(3) The "Get code" button:

```diff
-                className="shrink-0 rounded-lg bg-indigo-50 px-4 py-2.5 text-sm font-medium text-indigo-600 transition hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-50"
+                className="shrink-0 rounded-lg bg-indigo-500/20 px-4 py-2.5 text-sm font-medium text-indigo-200 transition hover:bg-indigo-500/30 disabled:cursor-not-allowed disabled:opacity-50"
```

(4) Confirm that the "selected state" of both tab buttons is still `bg-white text-neutral-900 shadow-sm` (keep the white selected block; no change needed).

- [ ] **Step 2: Build verification**

Run: `cd frontend && npm run build`

Expected: build passes.

- [ ] **Step 3: Commit**

```bash
git add "src/app/(auth)/login/page.tsx"
git commit -m "style: dark theme tweaks on login page"
```

---

### Task 3: Add Phone/Email Tabs + Phone Registration to the Register Page

**Files:**
- Modify: `frontend/src/app/(auth)/register/page.tsx`

**Interfaces:**
- Consumes:
  - `authAPI.sendSMSCode(phone)` (`@/lib/api`)
  - `authAPI.loginByPhone(phone, code)` → `{ token, user }` (the backend auto-registers)
  - `authAPI.updateMe(token, { name })`
  - `setToken` / `setUser` (`@/lib/auth`)
  - `AuthCard` / `FormField` / `inputBase` / `fieldState` / `PasswordInput`
- Produces: the register page supports phone registration (phone tab by default) and email registration

- [ ] **Step 1: Rewrite register/page.tsx**

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

  // Phone registration
  const [phoneName, setPhoneName] = useState("");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [countdown, setCountdown] = useState(0);
  const [phoneNameError, setPhoneNameError] = useState("");
  const [phoneError, setPhoneError] = useState("");
  const [codeError, setCodeError] = useState("");

  // Email registration
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

  const handlePhoneRegister = async (e: React.FormEvent) => {
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
      // Nickname: use the entered one if provided, otherwise default to "User + last 4 digits of the phone number"
      const nickname = phoneName.trim() || `User${phone.slice(-4)}`;
      if (!data.user.name) {
        await authAPI.updateMe(data.token, { name: nickname });
      }
      setToken(data.token);
      setUser({ ...data.user, name: data.user.name || nickname });
      toast.success("Registration successful!");
      router.push("/dashboard");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  const handleEmailRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    const errs: typeof emailErrors = {};
    if (!emailName.trim()) errs.name = "Please enter a nickname";
    if (!EMAIL_RE.test(email)) errs.email = "Please enter a valid email address";
    if (password.length < 6) errs.password = "Password must be at least 6 characters";
    if (confirm !== password) errs.confirm = "The two passwords do not match";
    setEmailErrors(errs);
    if (Object.keys(errs).length > 0) return;

    setLoading(true);
    try {
      const data = await authAPI.registerByEmail(email, password, emailName.trim());
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
          <Link href="/login" className="font-medium text-indigo-300 hover:text-indigo-200">
            Log in now
          </Link>
        </>
      }
    >
      {/* Phone / email segmented toggle */}
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
          Phone registration
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
          Email registration
        </button>
      </div>

      {tab === "phone" ? (
        <form onSubmit={handlePhoneRegister} className="space-y-4" noValidate>
          <FormField id="pname" label="Nickname" error={phoneNameError}>
            <input
              id="pname"
              type="text"
              value={phoneName}
              onChange={(e) => setPhoneName(e.target.value)}
              className={`${inputBase} ${fieldState(!!phoneNameError)}`}
              placeholder="Your nickname (optional)"
            />
          </FormField>
          <FormField id="pphone" label="Phone number" error={phoneError}>
            <input
              id="pphone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, "").slice(0, 11))}
              className={`${inputBase} ${fieldState(!!phoneError)}`}
              placeholder="Enter an 11-digit phone number"
              required
            />
          </FormField>
          <FormField id="pcode" label="Verification code" error={codeError}>
            <div className="flex gap-3">
              <input
                id="pcode"
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
                className="shrink-0 rounded-lg bg-indigo-500/20 px-4 py-2.5 text-sm font-medium text-indigo-200 transition hover:bg-indigo-500/30 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {countdown > 0 ? `${countdown}s` : codeSent ? "Resend" : "Get code"}
              </button>
            </div>
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "Signing up..." : "Sign up"}
          </button>
        </form>
      ) : (
        <form onSubmit={handleEmailRegister} className="space-y-4" noValidate>
          <FormField id="name" label="Nickname" error={emailErrors.name}>
            <input
              id="name"
              type="text"
              value={emailName}
              onChange={(e) => setEmailName(e.target.value)}
              className={`${inputBase} ${fieldState(!!emailErrors.name)}`}
              placeholder="Your nickname"
              required
            />
          </FormField>
          <FormField id="email" label="Email" error={emailErrors.email}>
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
          <FormField id="password" label="Password" error={emailErrors.password}>
            <PasswordInput
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={fieldState(!!emailErrors.password)}
              placeholder="At least 6 characters"
              required
              minLength={6}
            />
          </FormField>
          <FormField id="confirm" label="Confirm password" error={emailErrors.confirm}>
            <PasswordInput
              id="confirm"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              className={fieldState(!!emailErrors.confirm)}
              placeholder="Enter the password again"
              required
            />
          </FormField>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-indigo-600 py-2.5 text-sm font-medium text-white transition hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loading ? "Signing up..." : "Sign up"}
          </button>
        </form>
      )}
    </AuthCard>
  );
}
```

- [ ] **Step 2: Build verification**

Run: `cd frontend && npm run test && npm run build`

Expected: 18/18 tests pass; build passes.

- [ ] **Step 3: Manual verification (dev mode, backend must be running)**

```bash
cd frontend && npm run dev
```

- [ ] Open `/register`: starfield background + transparent glass card; phone tab by default; switching to the email tab works
- [ ] Phone registration: enter phone number → get verification code (visible in the dev backend log as `📱 [DEV] ... Code:`) → enter the code + nickname → registration succeeds and redirects to `/dashboard`, with the nickname shown in the sidebar
- [ ] Register with an empty nickname: the nickname automatically becomes "User + last 4 digits"
- [ ] Register again with an already-registered phone number: it logs in directly and does not overwrite the existing nickname
- [ ] `/login`: starfield background + glass card; both phone and email login work; password show/hide works
- [ ] The starfield background stays fixed when scrolling on a small screen
- [ ] No errors in the console

- [ ] **Step 4: Commit**

```bash
git add "src/app/(auth)/register/page.tsx"
git commit -m "feat: phone registration on register page (sms code + nickname)"
```

---

## Self-Review Record

- **Spec coverage**: starfield background (Task 1 AuthCard), transparent glass card (Task 1), dark form components (Task 1), login page dark tweaks (Task 2), phone registration tab + flow + default nickname (Task 3), zero backend changes (plan constraint), error-handling toasts (every task).
- **Placeholder scan**: no TBD/TODO; the complete code for every change is provided.
- **Type consistency**: the signatures of `authAPI.sendSMSCode` / `loginByPhone` / `updateMe` / `setToken` / `setUser` / `AuthCard` / `FormField` / `inputBase` / `fieldState` / `PasswordInput` match the existing code; Task 3 consumes Task 1's components, so no import changes are needed.
