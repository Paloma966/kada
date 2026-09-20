# Kada Design

Kada is a short-link management and analytics platform. Users create short links, organise them, and
watch how they are clicked. This document explains how the system is put together and why.

- [1. Product summary](#1-product-summary)
- [2. System overview](#2-system-overview)
- [3. Repository layout](#3-repository-layout)
- [4. Backend](#4-backend)
- [5. Data model](#5-data-model)
- [6. Frontend](#6-frontend)
- [7. Key flows](#7-key-flows)
- [8. Cross-cutting decisions](#8-cross-cutting-decisions)
- [9. Configuration](#9-configuration) 
- [10. Deployment](#10-deployment)
- [11. Testing](#11-testing)
- [12. Known limitations](#12-known-limitations)

## 1. Product summary

| Capability | Notes |
|---|---|
| Authentication | Phone + SMS code behind a graphical challenge, JWT bearer token for the API |
| Links | Create, edit, disable, expire, password protect, batch delete and batch tag |
| Organisation | Folders, tags, workspaces |
| Custom domains | DNS TXT ownership verification before a domain can serve links |
| Analytics | Click totals, daily trend, platform breakdown, visitor list, raw event log |
| Integrations | UTM templates, QR codes, CSV export, personal API tokens |
| AI assistant | RAG over the project knowledge base, tools that run as the signed-in user |
| Presentation | Bilingual UI (Chinese / English), light and dark themes |

The primary tension in the design is that **redirection is hot and analytics is cold**. Following a
short link must be fast and must not fail because a database is busy, while counting clicks is
allowed to lag by a second. The architecture is split along that line.

## 2. System overview

```text
                       ┌────────────────────────────────────────────┐
   browser ──────────► │ nginx :80                                  │
                       │  /api/  and  /r/  ──► backend             │
                       │  everything else  ──► frontend            │
                       └───────┬───────────────────────┬────────────┘
                               │                       │
                    ┌──────────▼─────────┐   ┌─────────▼──────────┐
                    │ backend (Go/Gin)   │   │ frontend (Next.js) │
                    │  :8080             │   │  :3000             │
                    └──┬───────┬─────┬───┘   └────────────────────┘
                       │       │     │
          ┌────────────▼─┐ ┌───▼───┐ │
          │ PostgreSQL   │ │ Redis │ │ publish click
          │ (source of   │ │ cache │ │
          │  truth)      │ │ + rate│ │
          └──────▲───────┘ │ limit │ │
                 │         └───────┘ │
                 │                   ▼
                 │            ┌─────────────┐
                 └────────────│   Kafka     │
                    consume   │  topic:     │
                 ┌────────────│  clicks     │
                 │            └─────────────┘
        ┌────────▼─────────┐
        │ worker (Go)      │
        │ :no http port    │
        └──────────────────┘
```

Components:

| Component | Process | Responsibility |
|---|---|---|
| `cmd/server` | `backend` | HTTP API, short-link redirection, publishes click events |
| `cmd/worker` | `kafka-worker` | Consumes the `clicks` topic and persists click rows |
| `cmd/migrate` | one-off | Applies the database schema (GORM AutoMigrate) |
| `frontend` | `frontend` | Next.js App Router UI; talks to the API over HTTP only |
| `postgres` | `postgres` | Source of truth; also the analytics store |
| `redis` | `redis` | Short-code cache and rate-limit counters |
| `kafka` | `kafka` | Durable buffer between request handling and click persistence |
| `nginx` | `nginx` | Single entry point, routes `/api/` and `/r/` to the API |

There is exactly one writer of the database schema (see
[4.4 Schema ownership](#44-schema-ownership)) and no service talks to another service's database.
The frontend never touches PostgreSQL or Redis.

## 3. Repository layout

```text
backend/
  cmd/server/          API entry point: wiring, routes, graceful shutdown
  cmd/worker/          Kafka consumer entry point
  cmd/migrate/         schema application
  config/              environment parsing (single source of configuration)
  internal/
    domain/            transport models (JSON tags, request validation)
      entity/          GORM models == the database schema
    handler/           HTTP layer, one package per resource
    service/           business logic and persistence
    infra/             database, redis, SMS, URL safety, link preview, user-agent
    middleware/        JWT auth, rate limiting, real client IP
    mq/                Kafka publisher and event payloads
  db/                  (removed) schema used to live here as SQL migrations
frontend/
  src/app/             routes: landing, (auth), dashboard/**, r/[code]
  src/components/      shared UI components
  src/lib/             API client, auth storage, i18n, canvas/starfield helpers
docs/                  this documentation
deploy/                server bootstrap scripts and systemd units
nginx/                 reverse proxy configuration
scripts/               maintenance scripts
```

## 4. Backend

### 4.1 Layering

```text
HTTP request
   │
   ▼
middleware        JWT validation, API-token validation, rate limiting, client IP
   │
   ▼
handler/<res>     bind + validate JSON, map service errors to HTTP status
   │  depends on a small interface it declares itself, so it can be tested with a fake
   ▼
service/<res>     business rules, ownership checks, transactions, cache invalidation
   │  depends on *gorm.DB
   ▼
domain/entity     database rows
```

Rules that keep this honest:

- A handler never touches the database. It declares the slice of the service it needs as an
  interface (`handler/auth.AuthService`), which is why handler tests need no database.
- A service never touches `*gin.Context`. It takes `context.Context` plus plain values.
- Ownership is enforced in the service by putting `user_id` into the `WHERE` clause, never by
  trusting an ID that arrived from the client.
- Errors returned by services are safe to show to a user. Internal details stay in the logs.

### 4.2 Request path

`cmd/server/main.go` is the only place where the object graph is built. It creates the database
handle, Redis client, SMS sender, cache service, Kafka publisher, then every service, then every
handler, and finally registers routes:

```go
v1 := r.Group("/api")
authH.RegisterRoutes(v1, authMW, strictMW)
linkH.RegisterRoutes(v1, authMW)
// ...
```

Each handler owns its own route table (`RegisterRoutes`), so adding a resource means adding one
package and one line in `main.go`, not editing a central route list.

Public endpoints that do not require a session (`/api/auth/*`, `/r/:code`) are registered inside the
handler without the auth middleware; everything else is grouped behind it.

The Python AI service is the one exception, and it is not registered here at all: `/api/ai/*` is a
reverse proxy (`internal/handler/ai`) that authenticates the caller, injects the user id, and forwards
to the AI service with its own mount prefix stripped (`/api/ai/conversations/current` ->
`/conversations/current`). The Python service owns its own paths and knows nothing about `/api/ai`, so
the two surfaces can move independently - and, as everywhere else in this project, no `/v1` appears in a
URL: the `v1` in the snippet above is a variable name.

### 4.3 Short-link resolution

Redirection has its own hot path that deliberately avoids PostgreSQL:

```text
GET /r/:code
   │
   ├─ Redis cache hit ──► serve
   │
   └─ miss ──► SELECT ... WHERE short_code = $1 AND is_active
                 │
                 ├─ expired?      ──► error page
                 ├─ has password? ──► password page / verify
                 ├─ platform needs a guide page? ──► intermediate page with deeplinks
                 └─ otherwise     ──► 302 to the target
```

The result is written back to Redis with a TTL, and every mutation (update, delete, batch delete,
workspace delete) invalidates the affected short codes. This is a cache-aside pattern; PostgreSQL
remains the source of truth and a cold cache only costs latency.

### 4.4 Schema ownership

The schema is defined by the GORM models in `internal/domain/entity` and applied with
`AutoMigrate`. There are no SQL migration files.

- `cmd/migrate` applies the schema (`cd backend && go run ./cmd/migrate/`).
- The API server applies it on startup **only** when `DB_AUTO_MIGRATE=true`, so that in production a
  restart cannot change the database as a side effect.
- AutoMigrate is additive: it creates missing tables, columns, indexes and constraints, and never
  drops anything. That makes it safe on an existing installation, and it means a rollback to an
  older binary leaves the newer columns in place.
- Because the models are the only definition, a column that exists in code but not in the database
  (or the reverse) cannot happen.

Raw SQL is still used where a single statement is the point — the atomic verification-code consume,
the idempotent click insert, bulk insert-selects — and each of those sites is commented as such.

### 4.5 Error mapping

| Situation | HTTP |
|---|---|
| Invalid or missing body | 400 |
| No or invalid token | 401 |
| Code or password rejected | 401 |
| Not the owner / duplicate resource | 403 / 409 |
| Unknown resource | 404 |
| Rate limited | 429 |
| Anything unexpected | 500, with a generic message |

Services return sentinel errors for the cases a handler must distinguish (for example
`domain.ErrEmailTaken`), and wrap everything else with context so the log explains the failure while
the response does not.

## 5. Data model

```text
users ──┬── links ──┬── click_logs
        │           └── link_tags ── tags
        ├── folders ──┘ (links.folder_id)
        ├── workspaces ─┘ (links.workspace_id)
        ├── domains
        ├── utm_templates
        └── api_tokens

sms_codes       (standalone, keyed by phone)
login_captchas  (standalone, keyed by id)
```

| Table | Purpose | Notable columns |
|---|---|---|
| `users` | Accounts | `phone` (unique); `email`, `password_hash`, `wechat_openid` are retained legacy columns that no sign-in path reads |
| `links` | Short links | `short_code` (unique), `original_url`, `domain`, `password_hash`, `expires_at`, `is_active`, `click_count` |
| `click_logs` | One row per click | `platform`, `ip`, `referer`, `event_id` (unique, idempotency) |
| `folders`, `tags`, `link_tags` | Organisation | `link_tags` is a plain junction table |
| `workspaces` | Team/project grouping | `slug` (unique), `links.workspace_id` |
| `domains` | Custom domains | `verified`, unique per `(user_id, name)` |
| `utm_templates` | Reusable UTM sets | `utm_*` columns |
| `api_tokens` | Programmatic access | `token_hash` (unique); the raw token is shown once |
| `sms_codes` | Pending SMS logins | `code_hash` (sha256), `ip` (per-IP quotas), `attempts`, `expires_at`; the plaintext code is never stored |
| `login_captchas` | Pending graphical challenges | `code_hash` (sha256 of the upper-cased answer), `attempts`, `expires_at` |

Design rules:

- **Passwords are never stored in a form that can be replayed.** Account and link passwords use
  bcrypt; SMS codes, captchas and API tokens are stored as SHA-256 hashes, and the plaintext of a
  one-time secret is never written to the database or the log.
- **The graphical challenge lives in PostgreSQL, not Redis.** Redis is optional in this deployment - the
  rate limiter fails open when it is down - and a captcha that silently stops being enforced is worse than
  no captcha, because the endpoint still looks protected. The database is not optional.
- **Deletion semantics are explicit.** Deleting a user cascades to their folders, tags, domains,
  templates and tokens, but only detaches links (`ON DELETE SET NULL`) so public links do not vanish.
  Deleting a folder or workspace detaches its links. Deleting a link cascades to its click logs.
- **Timestamps are UTC** (`timestamptz`); the frontend formats for display.

## 6. Frontend

Next.js App Router with React Server Components only where they help; the dashboard is client-side
because it is highly interactive.

| Path | Content |
|---|---|
| `/` | Landing page with the animated starfield |
| `/login` | Sign-in: phone number, graphical challenge, SMS code, privacy consent |
| `/register` | Redirects to `/login`: a new phone number is registered on the spot |
| `/privacy` | Privacy policy, linked from the consent checkbox before sign-in |
| `/dashboard` | Overview: totals, trend, recent links |
| `/dashboard/links`, `/new`, `/[id]` | Link list, creation form, detail/edit |
| `/dashboard/analytics`, `/events`, `/customers` | Click analytics, raw events, visitors |
| `/dashboard/folders`, `/tags`, `/domains`, `/utm`, `/settings` | Organisation and account settings |
| `/r/[code]` | Client-side fallback page for a short code |

Data fetching uses SWR against `src/lib/api.ts`, which is the single place where the API surface is
declared and where the bearer token is attached. Components never call `fetch` directly.

State that must survive a reload (session token, profile, locale) lives in `localStorage` behind
helpers in `src/lib/auth.ts` and `src/lib/i18n`.

### 6.1 Internationalisation

The UI ships in Chinese and English. The dictionary is keyed by the **Chinese source string**:

```tsx
const t = useT();
return <button>{t("创建链接")}</button>;
```

- `zh` is the identity map, so an untranslated key degrades to readable Chinese rather than a blank.
- `en` holds the translation; a missing key falls back to the key itself.
- The active locale is stored under `kada.locale`. `useT()` returns a stable function, and locale
  changes re-render through the provider.
- Server-rendered output always starts in `zh` because `localStorage` is unreadable during SSR; the
  provider switches after hydration.

When adding UI text: write the Chinese string, wrap it in `t()`, and add the English entry to
`dictionary.ts`. The dictionary is the only place English copy lives.

### 6.2 Theme

The app follows the browser's `prefers-color-scheme` by default and lets the user override it from
Settings → 外观, with a third option that hands control back to the system.

- `data-theme="light|dark"` on `<html>` is the single source of truth. Tailwind's `dark:` variant is
  redefined as an attribute selector, so it works on any browser and the OS setting and an explicit
  choice take the same path.
- **Colours are roles, not shades.** `text-body` and `bg-canvas` resolve per theme; `text-gray-700`
  cannot, because a shade that reads correctly on white is usually wrong on near-black. The scales are
  not inverted wholesale: brand and status colours are fixed, because `bg-indigo-600 text-white` is a
  pair and brightening the indigo for a dark canvas would leave white text on a light blue.
- **A tint is a surface role and both halves move together.** `bg-brand-soft text-brand-ink` is a chip, a
  selected nav item, an icon tile - not a fixed pair - so it is redefined per theme as a unit. Theming
  only one of the two is how a light chip ends up carrying light ink.
- **`gray-50` is the one raw Tailwind neutral remapped for dark mode.** The app uses it purely as a
  neutral surface (`bg-gray-50`, `bg-gray-50/50`, `border-gray-50`, `hover:bg-gray-50`), always under
  `text-*` classes that are themselves themed; left alone it paints white panels and near-white code
  chips on a near-black page. `frontend/scripts/check-theme-css.mjs` asserts both this and the fact that
  `indigo-600`/`red-600` are *not* remapped, so the two rules cannot be confused by accident.
- The sign-in and landing screens are dark in **both** themes. Anything on them that must stay light -
  the selected segment, the language chip - uses literal white rather than a themed surface.
- **The switch is in the top bar.** A one-click sun/moon toggle sits beside the language switcher and pins
  an explicit choice; Settings keeps the three-way control (follow system / light / dark). Before
  hydration the toggle renders the server's assumption and is disabled, because the real value comes from
  `localStorage` and reading it during the first client render is the hydration mismatch `useHydrated`
  exists to prevent.
- An inline `<script>` in the document head sets the attribute while the HTML is parsed. It is
  deliberately **not** `next/script`'s `beforeInteractive`: that queues the body through Next's loader,
  which measured 62ms *after* the first frame and produced a visible light flash. `scripts/check-theme-timing.mjs`
  measures this and fails if the theme lands after the first frame.

Client components that read `localStorage` gate on `useHydrated()` (`src/lib/useHydrated.ts`). Reading a
token or a stored preference straight into render output makes the server's HTML and the client's first
render disagree, and React discards the tree as a hydration mismatch.

## 7. Key flows

### 7.1 Sign-in is phone-only

There is one way in: a phone number and an SMS code. A number that has never been seen is registered on
the spot, so "sign in" and "sign up" are the same request and there is no separate registration flow.

Two other sign-in paths were removed rather than hidden, and the reason is worth recording:

- **WeChat.** The schema had carried `wechat_openid`/`wechat_unionid` from the start and the config had
  `WECHAT_APP_ID`/`WECHAT_APP_SECRET`, but no route ever read them. Implementing it needs an approved
  WeChat Open Platform application, and an individual cannot register one, so the columns stay (existing
  data is untouched) and nothing else does.
- **Email + password.** It duplicated what the phone path already did, added a second credential to
  police, and after the phone flow was hardened it was the only endpoint left that could be attacked
  without an SMS cost. `users.email` and `users.password_hash` remain in the schema so old rows keep
  their data; `email` is now an optional contact field on the profile and nothing reads `password_hash`.

### 7.2 Login by phone

```text
GET /api/auth/captcha
  → generate a 4-character code and its SVG (internal/infra/captcha, no image or font dependency)
  → store sha256(NORMALISED code) in login_captchas with a 5-minute expiry
  → 200 { captcha_id, image: "data:image/svg+xml;base64,..." }   (Cache-Control: no-store)

POST /api/auth/send-sms-code   { phone, captcha_id, captcha_code }
  → one UPDATE consumes the challenge atomically (unused, unexpired, attempts < 5)
      failure → 400, and attempts is incremented for that id
  → per-phone quotas: 60s cooldown, 10/day            → 429 + Retry-After
  → per-IP quotas: 10/hour, 30/day                    → 429
  → provider sends the code, sha256(code) is stored with a 5-minute expiry
      provider error → the provider code and message are surfaced (see 8.4)
  → 200

POST /api/auth/login-by-phone
  → one UPDATE consumes the pending code atomically (unused, unexpired, attempts < 5)
      failure → increment attempts for the phone, return 401
  → find or create the user by phone
  → sign JWT → 200 { token, user }
```

Three properties of that ordering are deliberate:

1. **The captcha is checked first and is burned either way.** An unauthenticated endpoint that makes the
   server send paid messages is the most abusable thing in this app, so no send can happen without a human
   having solved a challenge - and a challenge that survives a refused send is a challenge an automated
   caller can reuse. The client re-fetches one after every attempt for exactly that reason.
2. **The quotas are independent.** The per-phone pair bounds the damage to one victim; the per-IP pair is
   what notices a script walking a list of numbers, which the per-phone quotas never would (each number is
   used once). The per-IP figures are looser on purpose: a campus or office NAT legitimately shares an
   address.
3. **A quota is a 429, a wrong captcha is a 400.** Returning 500 for both - which the handler used to do -
   tells a client to retry a request that will keep failing, and tells an operator to look for a server
   fault that does not exist.

### 7.3 Recording a click

```text
GET /r/:code
  → resolve the link (cache, then database)
  → build a click event with a random event_id
  → publish to Kafka
      publish fails → write the click row directly (degraded mode, same event_id)
  → redirect the visitor

worker: FetchMessage → WriteClick → CommitMessages
  INSERT click_logs ... ON CONFLICT (event_id) DO NOTHING
      inserted → UPDATE links SET click_count = click_count + 1
      conflict → commit only (the click was already counted)
```

Kafka is at-least-once, so `event_id` is what makes counting exact. The consumer commits offsets
only after a successful write, and a poison message (invalid JSON, foreign-key violation) is dropped
after three attempts rather than blocking the single-partition consumer group forever.

## 8. Cross-cutting decisions

### 8.1 Configuration

One `Config` struct in `backend/config` is parsed once from the environment; nothing else reads
`os.Getenv`. Missing values fall back to development defaults, except in release mode where weak JWT
secrets are rejected outright. `backend/.env.example` and the root `.env.example` document every
variable.

### 8.2 Caching and degradation

Redis is used for the short-code cache and for rate-limit counters, and both **fail open**: if Redis
is unreachable the API keeps serving, caching is skipped, and rate limiting stops rejecting requests
until the client reconnects. Availability of the product is ranked above strictness of the limiter.

### 8.3 Rate limiting

Sliding-window counters in Redis, in three tiers: a global limit for normal API traffic, a strict
limit for auth endpoints, and a higher-throughput limit for redirection. The client IP comes from
`X-Real-IP`, which nginx rewrites. `X-Forwarded-For` is not trusted, because a client can forge it.

### 8.4 Third-party failures are made visible

An SMS provider rejection (unapproved signature, unapproved template, disabled AccessKey) is a
configuration mistake, not a transient failure. The provider's error code and message are logged and
- outside release mode - returned to the caller, because a generic "please try again later" makes the
configuration impossible to diagnose. Internal details are suppressed in release mode.

### 8.5 Security posture

- JWT: HS256 only, expiry required, secret strength enforced in release mode.
- Link targets: only `http` and `https` are accepted, which also blocks `javascript:` and `data:`
  payloads from the guide page.
- Link preview: the fetcher refuses internal networks and cloud metadata addresses (SSRF).
- CSV export: fields are escaped and formula-injection prefixes are neutralised.
- Error responses never echo internal errors.
- Containers run as a non-root user; nginx sets the usual response security headers.
- The AI service refuses its own paths (`/chat`, `/conversations/*`) unless the request carries the
  gateway's shared secret, so being able to reach `127.0.0.1:8000` is not the same as being allowed to
  use it (see 8.6).

### 8.6 AI tools act as the signed-in user

The AI service holds no credential of its own. The Go gateway already forwards the caller's
`Authorization` header, and the tools that read link statistics or create short links pass it on to the
existing `/api/*` routes. Four consequences decide the shape of that:

- **Permission checks live in exactly one place.** Python neither parses nor validates the JWT, it only
  forwards it. A disabled account or an expired token fails a tool call exactly as it fails the user's own
  call, and nothing has to be restarted when that happens.
- **The user id is an assertion by the gateway, and only because of the secret.** The gateway drops any
  client-supplied `X-Kada-User-ID` and sets it from the verified JWT; the service trusts that header only
  while the same request also carries `AI_INTERNAL_SECRET`, a value no other local process knows. Without
  that second half, any process on the host could read or delete another user's conversations - which is
  why the header itself was never the weak point worth changing.
- **The credential is per request, not per process.** It travels in a `ContextVar` that the chat route
  binds before generating, so two users served by the same worker cannot see each other's token. That is
  also why these tools cannot live in the MCP subprocess: stdio is a single long-lived session shared by
  every request, and `langchain-mcp-adapters` supports per-call headers only on HTTP transports. MCP stays
  for tools that need no user identity.
- **A failed tool degrades the answer, not the request.** Tools return their failure as text (an HTTP
  status, a missing credential) so the model can explain it, in the same spirit as RAG retrieval falling
  over to "no reference material".

## 9. Configuration

| Variable | Default | Purpose |
|---|---|---|
| `DATABASE_URL` | local `kada` database | PostgreSQL DSN |
| `DB_AUTO_MIGRATE` | `true` | Apply the schema on API startup |
| `REDIS_URL` | `redis://localhost:6379` | Cache and rate limiting (optional) |
| `JWT_SECRET` | weak dev value | Token signing; must be strong in release mode |
| `JWT_EXPIRES_IN` | `720h` | Token lifetime |
| `PORT` | `8080` | API listen port |
| `GIN_MODE` | `debug` | `release` silences dev output and enables strict checks |
| `API_BASE_URL` | `https://kada.click` | Base for generated short URLs |
| `FRONTEND_URL` | `http://localhost:3000` | CORS and redirects |
| `SMS_ACCESS_KEY_ID`, `SMS_ACCESS_KEY_SECRET` | empty | Alibaba Cloud credentials; empty disables real sending (and, in production, sign-in) |
| `SMS_SIGN_NAME`, `SMS_TEMPLATE_CODE` | empty | The system-granted signature and template from the PNVS console; both are required, see below |
| `KAFKA_BROKERS` | empty | Comma-separated brokers; empty disables Kafka (clicks are written directly) |
| `KAFKA_TOPIC` | `clicks` | Click event topic |
| `NEXT_PUBLIC_API_URL` | `""` (same origin) | API base for the browser |
| `AI_BASE_URL` | `http://127.0.0.1:8000` | Internal Python AI service the gateway proxies `/api/ai/*` to |
| `AI_INTERNAL_SECRET` | empty | Injected as `X-Internal-Secret`; the AI service accepts only requests carrying it |

`SMS_SIGN_NAME` has no default on purpose. It used to fall back to the literal `kada`, and for this
deployment that is in fact the account's real signature - which is precisely what made the outage
invisible: with the signature always supplied by a default, the missing template code (see 9.1) still
produced a constructed sender, and the failure came back as an Aliyun rejection rather than as "this is
not configured". Empty now means "not configured", and the startup log says so. The cost is one explicit
setting: `SMS_SIGN_NAME=kada` for this account.

None of these secrets are set on the server by hand any more. The deploy job writes them from GitHub
repository secrets with `deploy/upsert-env.sh`, as its **first** step, before a single service is replaced.
That script draws the line between the two kinds of missing value, and the line is a product decision:

- `--require` for the AI keys. A service that starts, passes `/healthz` and then fails every question is
  worse to diagnose than a refused deploy, so the deployment fails while the previous build keeps serving.
- `--warn` for the SMS credentials. The Aliyun signature and template have to be approved in the console,
  which takes days, and blocking every deploy until then would stop unrelated fixes from shipping. The
  warning is doubled: a runner-side step annotates the run and writes the job summary, and the server's own
  log repeats it. It is a warning about the *deployment*, not about the product - phone + SMS code is the
  only way to sign in, so a site without them is up and unusable, and in release mode the code is never
  logged.

The Python AI service reads its own environment (`backend/ai/app/config.py`), and in production those
values live in `/opt/kada/ai/ai.env` rather than in the repository's `.env`:

| Variable | Default | Purpose |
|---|---|---|
| `DEEPSEEK_API_KEY` | empty | Chat model key (DeepSeek, OpenAI-compatible API) |
| `aliyun` | empty | DashScope key for the RAG embeddings; the variable name is literally `aliyun` |
| `POSTGRES_URL` | `...@127.0.0.1:5432/kada_ai` | AI database: conversations plus the pgvector knowledge base |
| `REDIS_URL` | `redis://127.0.0.1:6379/0` | Session hot cache; optional, the service falls back to PostgreSQL |
| `KADA_API_BASE` | `http://localhost:8080` | Go API the business tools call back into |
| `AI_INTERNAL_SECRET` | empty | The same value as the gateway's; empty disables the inbound check |

> There is no AI service token. The business tools act as the signed-in user, with the JWT the gateway
> forwards on each chat request (see 8.6); the long-lived token this service used to hold, and the
> hardcoded fallback it carried, are both gone. `AI_INTERNAL_SECRET` is a different kind of value: it
> authenticates the *hop*, not a person. Leaking it is not enough to act on a user's links (that still
> needs their JWT), but it is enough to forge `X-Kada-User-ID` and read their conversations, so it is
> still a secret.

### 9.1 SMS verification

Sign-up by phone uses Alibaba Cloud **PNVS "SMS verification"** (`dypnsapi.aliyuncs.com`), not the
separate SMS product (`dysmsapi`). That distinction decides the configuration:

- It is the one SMS route open to **individually verified** accounts. The SMS product stopped accepting
  personal self-use qualifications, so personal signatures and templates can no longer be approved there.
- The account gets **one** system-granted signature and **one** system-granted template, to be taken from
  the PNVS console. They cannot be created or edited, and they must be used as a pair - a granted
  signature with a custom template is rejected, and so is the reverse.
- `SMS_SIGN_NAME` and `SMS_TEMPLATE_CODE` have **no defaults**. They used to fall back to a hardcoded
  signature and a made-up template code (`恒创联众` / `100001`, values that exist on nobody's account), which
  turned "nobody configured this" into a provider rejection that read like a broken account. Startup now
  names the missing setting instead, and phone sign-up stays disabled until it is set.
- The PNVS console, its data, and its package are all separate from the SMS product: sending is billed
  against a PNVS "SMS verification" package, which the SMS product's free trial does not cover.

**Where the working pair went (worth reading before debugging "SMS stopped working").**

SMS really did work in production once, and the reason it stopped is on the record:

1. `05eda08 feat: add Alibaba Cloud SMS verification` recorded the account's real pair in
   `backend/.env.example`: signature `kada`, and a template code of the form `SMS_…`. (The exact code is
   deliberately not repeated here - see the note below - but it is recoverable with
   `git show 05eda08:backend/.env.example`.)
2. `8420bd9 refactor: move the backend from pgx to GORM with AutoMigrate` blanked that whole SMS section of
   the example file while rewriting it, so **the only copy of the template code in the repository was
   deleted by an unrelated refactor**. Nothing failed at the time: the code still had the `恒创联众` /
   `100001` fallback and a `kada` default for the signature, so a deployment that had never configured the
   pair kept limping along.
3. `fbc1763 fix: fail on an unset SMS signature instead of substituting one` removed that fallback and made
   an unset signature or template a hard startup failure. Correct on its own - but by then the right value
   was gone from the repo, so "fail loudly" became "SMS is disabled and nobody knows what to put back".
4. Removing the `kada` default (this change) is the last step of the same idea. It costs one explicit
   setting - **the signature for this account is `kada`** - and buys a startup line that says what is
   missing instead of a provider error that looks like a broken account.

The template code and the AccessKey pair are **not in the repository and never were** (the AccessKey is
only ever read from the environment; a history search for the `LTAI` prefix finds nothing but accidental
substrings inside base64 hashes in `go.sum`). Both come from the Aliyun side: the pair from the PNVS
console, the credentials from RAM. That is also where they belong - a signature and a template code are
account-specific values, so they are kept in the environment (and, in production, in repository secrets
that the deploy writes out), not in the tree. Recording the incident above without re-committing the value
is the point: the lesson is "the value was lost", not "paste it back into the source".

## 10. Deployment

Three supported shapes:

1. **Docker Compose** (`docker compose up -d`): nginx, API, worker, AI, frontend, PostgreSQL, Redis and
   Kafka on one host. Suitable for a single server or local development.
2. **systemd + released binaries** (`deploy/`): the API and worker run as native processes, with
   PostgreSQL, Redis and Kafka provided separately. This is what the GitHub Actions deploy job uses:
   it builds the binaries, applies the schema with `bin/migrate`, restarts `kada-api`, and verifies
   the health endpoint, rolling back to the previous binary if the service does not come up.
3. **The AI service as a container of its own** (`deploy/docker-compose.ai.yml`), which is how it joins
   shape 2 on a host that already runs PostgreSQL and Redis. The root `docker-compose.yml` cannot be used
   for that: its `ai` service declares `depends_on`, so starting it would also start compose's own
   postgres and collide on `127.0.0.1:5432`. The dedicated file contains that one service and no
   dependencies at all. It runs with `network_mode: host` and binds `127.0.0.1:8000` - exactly where
   `AI_BASE_URL` already points - so the Go gateway reaches it without new wiring, and ufw keeps
   governing the port because it never enters Docker's iptables chains.

   The image is built **on the host**: a Python service is not a binary to copy, and only the source
   changes between deploys, so the pip layers stay cached. `deploy/deploy-ai.sh` is the only
   implementation of that step - the deploy job syncs `backend/ai`, `deploy/docker-compose.ai.yml` and
   the script, then runs it. It refuses to deploy without `/opt/kada/ai/ai.env`, or with an empty
   `DEEPSEEK_API_KEY` / `aliyun` in it (a service that starts and then fails every request is worse than a
   refused deploy), waits for `/healthz`, and rolls back to the previous image otherwise.
   `deploy/setup-ai-db.sh` creates `kada_ai` and enables the vector extension on a database that already
   exists, which `docker-entrypoint-initdb.d` can no longer do on an existing volume.

   The provider keys themselves reach the host from **GitHub repository secrets**, written into
   `/opt/kada/ai/ai.env` and `/opt/kada/backend/.env` by `deploy/upsert-env.sh` as the first step of the
   deploy, before any service is replaced. That file is the fix for a whole class of outage: both "the AI
   page is broken" and "the SMS code never arrives" turned out to be an empty line in a file on the
   server that nothing had ever checked. A `--require`d key that is missing fails the deployment while the
   previous build is still serving, and an empty `--set` leaves an existing hand-configured value alone.

The deploy job applies the schema **before** replacing the binary, so a failed migration leaves the
previous version running.

Verification happens in two places, which is deliberate:

- **On the host**, during the deploy step: `curl http://127.0.0.1:8080/api/health`. This is the
  authoritative check and the one that can fail the deploy. It depends on nothing but the API
  answering, so a TLS, DNS or network problem cannot be reported as a failed deployment.
- **From the runner**, afterwards: a smoke test of the public origin over HTTPS, its `/api/health`,
  the frontend, and the plain-HTTP entry point (which nginx redirects). It covers DNS, TLS and nginx as
  a visitor sees them, and it is **advisory**: a datacenter IP can be refused by the host's edge (this
  deployment answered `Connection reset by peer` during a TLS handshake while the same URL returned 200
  elsewhere), and gating a release on that would make green builds a matter of luck. Its result is
  reported as a warning and in the run summary, never as a failed deploy.

  The step absorbs a refusal rather than letting `curl` raise it. `continue-on-error` keeps it from
  failing the run, but on its own it still lets a non-zero `curl` surface as an error annotation and a
  red X next to a deployment that succeeded. The step therefore always exits 0, writes what happened to
  `verify-status.txt` and `verify-body.txt`, and the step after it reports those. A refusal is a fact to
  record, not an error to raise.

  Two shell details are load-bearing there, and `scripts/verify-deploy-step.test.js` runs this step
  against a stubbed `curl` to keep them honest: the exit status is read from a plain assignment, because
  `if ! curl` reports the status of the negation (a refusal recorded as `curl 0`) and `local rc=$?` reads
  the status of `local`; and only `/api/health` is asserted to carry a healthy payload, since the
  frontend serves HTML and asserting the same payload on it fails a perfectly healthy site.

The public origin defaults to `https://kada.click`. Override it with the `SITE_URL` **repository
variable** (Settings → Secrets and variables → Actions → Variables) when a deployment serves a
different name. It is a variable rather than a secret because the name is already public in the
certificate transparency logs, the DNS records and `nginx/nginx-prod.conf` - and because a deployment
step that requires manual setup is a failure mode of its own.

## 11. Testing

| Layer | Style |
|---|---|
| HTTP handlers | Table-driven tests with fake service implementations; no database |
| Services | Pure-function tests (validation, short-code generation, CSV escaping) |
| Schema | `internal/domain/entity` asserts table names, columns, unique indexes and delete rules against the GORM schema |
| Query shapes | Dry-run GORM sessions assert the generated SQL for the dynamic and bulk statements |
| Frontend | Vitest for pure helpers (starfield geometry, ophiuchus lines, utilities) |
| AI service | CI builds `backend/ai`, asserts the DashScope SDK is importable and boots the container against a real PostgreSQL to hit `/healthz` |

Run everything with `cd backend && go test ./... -count=1 -race` and `cd frontend && npm test`; CI
additionally runs `go vet`, `golangci-lint`,
`tsc --noEmit` and the production build on every pull request. The AI service gets a job of its own
(`ai-build`) because its failures show up at request time rather than at import time: a requirements.txt
missing the embedding SDK, or an image that cannot start, passes every startup check and only breaks when
a user asks a question.

## 12. Known limitations

These are deliberate trade-offs, not oversights:

- **Analytics are computed on demand.** `click_logs` is queried directly; there is no rollup table.
  Acceptable at the current volume, and the obvious first thing to change if the tables grow.
- **Kafka has one partition.** Click ordering is preserved and the throughput ceiling is a single
  consumer; a single-partition consumer group is also what makes the poison-message logic simple.
- **AutoMigrate cannot express every future change.** Renames, type narrowing and data backfills need
  a hand-written statement; there is no down-migration path.
- **No background job scheduler.** Expiry is evaluated at read time rather than by a sweeper.
- **Single-tenant deployment.** There is no organisation/role model beyond `workspaces`, which group
  links but do not grant access to other users.
- **Rate-limit state is per-Redis-instance**, so a multi-instance deployment needs a shared Redis
  (which is the intended topology).
