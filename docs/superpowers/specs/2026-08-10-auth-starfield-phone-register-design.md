# Starfield background on login/register pages + phone number registration design

Date: 2026-08-10
Status: confirmed with user

## Goals

1. Apply the homepage 3D starfield background to the login/register pages for visual consistency
2. Fix "cannot register with a phone number": add a phone number registration entry on the register page

## Scope

**Frontend (5 files changed)**
- `frontend/src/components/auth/AuthCard.tsx` — apply starfield background + transparent glass card
- `frontend/src/components/auth/FormField.tsx` — dark label / inputBase / fieldState
- `frontend/src/components/auth/PasswordInput.tsx` — dark input
- `frontend/src/app/(auth)/login/page.tsx` — dark segmented tab control, "get verification code" button style tweaks
- `frontend/src/app/(auth)/register/page.tsx` — add phone/email tab + phone number registration flow

**Backend: zero changes**

Backend capabilities already verified (no changes needed):
- `POST /auth/login-by-phone` (`backend/internal/service/auth_service.go`) — code login, auto-creates the account if the user is not found (auto-registration)
- `PATCH /me` (`backend/internal/handler/auth/handler.go:45`) + `api.ts:61 updateMe(token, { name })` — nickname can be written after authentication
- In dev mode the SMS verification code is printed to the backend log: `📱 [DEV] Phone: %s, Code: %s`

**Out of scope**: dashboard pages, `r/[code]`, any backend change, homepage (already done).

## Starfield background (AuthCard)

AuthCard changes from the light `bg-neutral-50` wrapper to:

```tsx
<div className="relative min-h-dvh bg-deep-space">
  <StarfieldCanvas className="fixed inset-0 h-full w-full" />
  <div className="relative z-10 flex min-h-dvh items-center justify-center px-4 py-12">
    {/* logo / title / card / footer */}
  </div>
</div>
```

- Reuse the `bg-deep-space` gradient + `StarfieldCanvas` (`fixed inset-0`, background stays fixed while scrolling)
- Card: **transparent glass** `rounded-xl border border-white/10 bg-white/[0.04] p-6 backdrop-blur-lg sm:p-8`
- Title in white; subtitle `text-indigo-200/80`; footer links `text-indigo-300`
- The logo block stays solid indigo

## Dark form components

**FormField**:
- label: `text-neutral-100`
- `inputBase`: `bg-white/10 border-transparent text-white placeholder:text-neutral-400/70 focus:ring-2`
- `fieldState`: invalid → `border-red-400/60 focus:ring-red-400/30`; normal → `border-white/15 focus:border-indigo-400 focus:ring-indigo-400/30`
- Error text: `text-red-400`

**PasswordInput**: same dark styling as inputBase; eye button `text-neutral-400 hover:text-neutral-200`

**login/page.tsx** inline style tweaks:
- Segmented tab container `bg-neutral-100` → `bg-white/10`; unselected text `text-neutral-300 hover:text-white`; selected block `bg-white text-neutral-900 shadow-sm` (keeps a white selected block)
- "get verification code" button `bg-indigo-50 text-indigo-600` → `bg-indigo-500/20 text-indigo-200 hover:bg-indigo-500/30`

## Phone number registration (register/page.tsx)

- The register page gains a **phone / email segmented tab**, defaulting to the phone tab (consistent with the login page)
- **Phone tab**:
  - Fields: nickname (optional) + phone number + verification code (with a 60s countdown)
  - Submit: validate 11-digit phone number, 6-digit code, nickname not required
  - Flow:
    1. `authAPI.sendSMSCode(phone)`
    2. `authAPI.loginByPhone(phone, code)` → backend auto-registers + returns token/user
    3. Nickname: `const nickname = name.trim() || \`user${phone.slice(-4)}\``
    4. If the returned `user.name` is empty → `authAPI.updateMe(token, { name: nickname })` to write the nickname
    5. `setToken/setUser` + redirect to `/dashboard`
  - An already registered phone number logs in directly (does not overwrite the existing nickname)
- **Email tab**: keeps the existing email registration (nickname + email + password + confirm), logic unchanged

## Error handling

- Keeps the existing toast + inline error pattern
- Failed code send, wrong code, failed login → toast shows the message returned by the backend

## Testing

- `npm run test` (existing 18 cases, expected unchanged)
- `npm run build` passes
- Backend: `go test ./...` (unchanged, regression check)
- Manual verification:
  - Login/register pages show the starfield background + transparent glass card, text is readable
  - Phone registration: enter a phone number → get verification code (visible in dev log) → enter code + nickname → registration succeeds and redirects to dashboard, nickname correct
  - Phone login still works; email registration/login still works
  - Starfield background stays fixed when scrolling on small screens
