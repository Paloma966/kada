# frontend homepage / login & register pages restyle design

date: 2026-08-05
status: confirmed with the user

## goals

restyle the three Kada frontend pages (homepage `/`, login `/login`, register `/register`) in a "reduce noise, don't swap the engine" way:

- keep the existing indigo multi-color system, drop gradients, lower visual noise, unify radius/border/shadow
- unify login/register into the same minimal shell, and close the gaps in the form experience
- don't touch backend APIs

## scope

- `frontend/src/app/page.tsx` (homepage)
- `frontend/src/app/(auth)/login/page.tsx`
- `frontend/src/app/(auth)/register/page.tsx`
- `frontend/src/app/globals.css` (design tokens)
- new `frontend/src/components/auth/`: `AuthCard`, `FormField`, `PasswordInput`

**out of scope**: the dashboard pages, the `r/[code]` redirect page, any backend change, the homepage quick-shorten tool (explicitly not wanted by the user).

## design tokens (globals.css)

- primary color `indigo-600` (#4f46e5), used only for buttons, links, logo, and selected states
- borders uniformly `neutral-200` hairline; backgrounds uniformly `neutral-50`
- two radius levels: buttons/inputs `rounded-lg` (8px), cards `rounded-xl` (12px); drop `rounded-2xl`
- one shadow level: cards `shadow-sm`; drop `shadow-lg` / `hover:shadow-lg`
- remove all `bg-gradient-to-*`

## shared components (new src/components/auth/)

- **AuthCard**: white card shell + consistent padding + brand logo + title + subtitle + bottom link area; shared by login/register
- **FormField**: label + input + inline error message (red text + red input border)
- **PasswordInput**: password field + show/hide toggle; reused by login and register

## homepage (page.tsx)

| section | current | change to |
|---|---|---|
| Header | white/80 + backdrop-blur + transparent border | sticky white background + `neutral-200` hairline bottom border; Logo + login (text) + register (solid) |
| Hero | purple gradient background, breathing-light badge, gradient-text title, large-shadow buttons | solid background; drop the badge; near-black title + a single indigo accent word; both buttons lose the large shadow |
| Features | 6 cards + `hover:shadow-lg` | keep the card structure; hover darkens the border; icon blocks uniformly on an `indigo-50` background |
| CTA | purple gradient background | solid `indigo-600` background, white text and white button |
| Footer | brand + copyright | keep it minimal |

## login page (page.tsx)

- use the `AuthCard` shell
- keep the phone/email tab switch, but turn it into a segmented control (gray background + white selected block)
- phone tab: phone number (validated as 11 digits) + verification code + "get code" countdown button (restyled, logic unchanged)
- email tab: email + `PasswordInput`
- unified inline errors + submit loading state

## register page (page.tsx)

- use the `AuthCard` shell
- fields: nickname, email, password, **confirm password (new)**
- validation: email format, password ≥6 chars, both passwords match, with inline red text
- register stays email-based (the backend has no phone registration endpoint)

## error handling

- inline form validation errors + toast on submit failure (reusing the existing sonner usage)
- button loading state reuses the existing `loading` state pattern

## testing

- `npm run lint`, `npm run build` pass
- manual verification: all three pages render, login/register form interaction, password show/hide, countdown, error messages
