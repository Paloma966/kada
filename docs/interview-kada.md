# Kada 短链接平台 — 面试通关讲解手册

> 适用岗位：后端（Go）/ 全栈
> 项目性质：前后端分离的短链接 SaaS 平台，含用户体系、短链管理、点击统计、自定义域名、多工作区、UTM 模板、开放 API。

---

## 目录

1. [一句话项目介绍（面试开场）](#一句话项目介绍)
2. [技术栈](#技术栈)
3. [整体架构](#整体架构)
4. [后端分层与目录结构](#后端分层与目录结构)
5. [数据库设计（10 张表）](#数据库设计)
6. [核心流程详解](#核心流程详解)
   - [认证：手机验证码登录](#认证手机验证码登录)
   - [短链创建](#短链创建)
   - [短链跳转 redirect](#短链跳转-redirect)
   - [点击事件异步链路（Kafka）](#点击事件异步链路-kafka)
7. [关键机制逐个拆解](#关键机制逐个拆解)
   - [缓存策略（Redis）](#缓存策略redis)
   - [限流（滑动窗口）](#限流滑动窗口)
   - [安全设计](#安全设计)
   - [平台检测与中间引导页](#平台检测与中间引导页)
8. [前端架构](#前端架构)
9. [部署与 CI/CD](#部署与-cicd)
10. [测试](#测试)
11. [面试题大全（含追问）](#面试题大全含追问)
12. [答不出来的兜底表达](#答不出来的兜底表达)

---

## 一句话项目介绍

> "Kada 是一个**短链接 SaaS 平台**。用户注册登录后可以创建、管理短链接，支持自定义域名、访问密码、过期时间、UTM 参数、文件夹/标签/工作区的分类管理；短链被点击后会做**平台检测**（微信/QQ/小红书等内置浏览器会渲染中间引导页），点击事件通过 **Kafka 异步削峰**写入数据库，并提供**数据看板**（总链接数、点击量、平台分布、每日趋势、访客明细）。技术栈是 **Go + PostgreSQL + Redis + Kafka + Gin**，前端 **Next.js + SWR + Tailwind**，用 **Docker Compose** 编排、**GitHub Actions** 做 CI/CD。"

---

## 技术栈

| 层 | 技术 | 用途 |
|----|------|------|
| 后端 | Go 1.26 + Gin | HTTP 服务 |
| 数据访问 | pgx v5（pgxpool 连接池）+ 原生 SQL | PostgreSQL 读写 |
| 数据库 | PostgreSQL 16 | 主存储 |
| 缓存/限流 | go-redis v9 + Redis 7 | 短码缓存、滑动窗口限流 |
| 消息队列 | segmentio/kafka-go + Kafka 3.8 | 点击事件异步落库 |
| 认证 | golang-jwt/v5（HS256）+ bcrypt | JWT + API Token |
| 短信 | 阿里云 dypnsapi SDK | 手机号验证码 |
| 前端 | Next.js 16（App Router）+ React 19 + TypeScript | 管理后台 |
| 前端数据 | SWR + react-hook-form | 请求缓存、表单 |
| 样式 | Tailwind CSS 4 | UI |
| 部署 | Docker Compose + Nginx + systemd | 容器化、反向代理、进程守护 |
| CI/CD | GitHub Actions | lint、test、build、deploy、release |
| 迁移 | golang-migrate | 数据库版本管理 |

> 面试点：为什么用 Go？Go 的并发模型（goroutine/select）、编译型语言的高性能、单二进制部署方便，非常适合做短链这种高并发、低延迟的后端服务。

---

## 整体架构

```
                         ┌────────────────────────────┐
  用户浏览器 ──────────► │  Nginx (80/443)            │
                         │  ├── /api/  ──► Go API     │
                         │  ├── /r/    ──► Go API     │
                         │  └── /      ──► Next.js    │
                         └────────────┬───────────────┘
                                      │
                 ┌────────────────────▼────────────────────┐
                 │        Go 后端（API, :8080）              │
                 │  ┌──────────────┐  ┌───────────────┐   │
                 │  │ Handler 层   │  │ Middleware     │   │
                 │  │  (Gin)       │  │ JWT/限流/IP    │   │
                 │  └──────┬───────┘  └───────┬───────┘   │
                 │         │                  │           │
                 │  ┌──────▼───────────────────▼──────┐   │
                 │  │ Service 层（业务逻辑）            │   │
                 │  └────────────┬─────────────────────┘   │
                 │            ┌──┴───────────┐             │
                 │       PostgreSQL          Redis         │
                 └────────────┬────────────────────────────┘
                              │ Kafka (topic: clicks)
                              ▼
                 ┌────────────────────────────┐
                 │   Go kafka-worker (消费)    │
                 │   click_logs 落库 + 计数    │
                 └────────────┬───────────────┘
                              ▼
                          PostgreSQL
```

### 关键设计决策（面试主动讲）

1. **模块化单体 + 独立 worker**：不是微服务，但把"点击事件消费"拆成独立的 `worker` 进程，与 API 共用一个代码库。好处是 API 不需要承担"读 Kafka + 写库"的耗时操作，点击高峰时 API 依然稳定。
2. **异步削峰**：短链跳转是**高频低延迟**场景，不可能每点一次就同步 `INSERT` + `UPDATE` 计数，那样会让跳转变慢。所以点击事件先发到 Kafka，由 worker 批量/顺序消费落库。
3. **降级路径**：Kafka 不可用时 `LogClick` 会**回退到直写数据库**（`ClickStore`），保证点击不丢失——体现"可用性优先"的工程思维。
4. **无 ORM，手写 SQL**：表结构复杂（多表 join、动态查询），手写 SQL 更可控、性能更好，也便于优化。面试可说"用 pgx 原生写 SQL 而非 GORM 是为了精确控制查询和避免 ORM 黑洞"。

---

## 后端分层与目录结构

```
backend/
├── cmd/
│   ├── server/main.go        # API 进程入口（装配依赖、注册路由）
│   ├── worker/main.go        # Kafka 消费者进程
│   └── migrate/main.go       # 数据库迁移
├── config/config.go          # 环境变量配置
├── db/migrations/            # 10 个迁移文件
├── internal/
│   ├── domain/models.go      # 领域模型/请求响应结构体（唯一"模型"层）
│   ├── handler/              # HTTP 层（auth/link/redirect/domain/folder/tag/utm/token/workspace/analytics）
│   ├── service/              # 业务逻辑层
│   ├── infra/                # 基础设施（database/redis/sms/preview/ua/urlcheck）
│   ├── middleware/           # JWT 认证、限流、真实 IP
│   └── mq/                   # Kafka 消息封装
└── go.mod
```

**经典的分层调用链**（面试最常考）：

> `Handler(HTTP) → Middleware(认证/限流) → Service(业务) → Infra/DB(存储)`

这套结构的好处：
- **职责单一**：Handler 只做参数解析和 HTTP 响应，Service 只做业务逻辑，Infra 只做技术细节。
- **便于测试**：Handler 依赖接口（如 `AuthService`、`LinkService` 接口），可以 mock；Service 依赖 `ClickWriter`、`SMSSender` 等接口也便于 mock。
- **依赖注入**：通过在 `main.go` 里"手工装配"依赖（new service → new handler），没有引入繁重的 DI 框架。

**main.go 装配示例**：
```go
// 1. 加载配置
cfg := config.Load()
// 2. 连接数据库
db, err := infra.NewDB(cfg.DatabaseURL)
// 3. 连接 Redis（ping 失败也保留客户端，限流 fail-open）
redisClient, err := infra.NewRedis(cfg.RedisURL)
// 4. 初始化短信（未配置则验证码打日志）
var smsSender service.SMSSender
if cfg.SMSAccessKeyID != "" { smsSender, _ = sms.NewAliyunSender(...) }
// 5. Kafka 发布者（无 broker 返回 nil = 禁用）
kafkaPub := mq.NewKafkaClickPublisher(cfg.Brokers(), cfg.KafkaTopic)
// 6. 组装 Service → Handler
linkSvc := service.NewLinkService(db, cfg.BaseURL, cacheSvc, kafkaPub, service.NewClickStore(db))
linkH := linkHandler.NewHandler(linkSvc)
// 7. 注册路由 + 启动
r := gin.Default()
r.GET("/api/health", ...)
redirectH.RegisterRoutes(r, redirectMW)
v1 := r.Group("/api")
authH.RegisterRoutes(v1, authMW, strictMW)
...
r.Run(":" + cfg.Port)
```

> 面试点：为什么要"接口 + 注入"？为了**可测试性**——单测里可以传入 fake 的 `ClickWriter`/`SMSSender`，不依赖真实 Redis/Kafka/短信网关。

---

## 数据库设计

数据库共 **10 张表**（10 个迁移文件）。用 `golang-migrate` 管理，每次变更一个 `.up.sql` + `.down.sql`，并维护 `schema_migrations` 状态表。

| 表 | 作用 | 关键字段 |
|----|------|----------|
| `users` | 用户（手机号/邮箱/微信多种登录）| `phone`,`email`,`wechat_openid`,`password_hash` |
| `links` | 短链核心表 | `short_code`(UNIQUE),`original_url`,`domain`,`password_hash`,`expires_at`,`click_count`,`user_id`,`workspace_id`,`folder_id`,`utm_*`,`ios_url`,`android_url` |
| `click_logs` | 点击日志 | `link_id`,`ip`,`user_agent`,`platform`,`referer`,`country/province/city` |
| `sms_codes` | 短信验证码 | `phone`,`code`,`used`,`expires_at`,`attempts` |
| `folders` | 文件夹 | `user_id`,`name` |
| `tags` | 标签 | `user_id`,`name`,`color` |
| `link_tags` | 链接-标签多对多 | `link_id`,`tag_id` |
| `domains` | 用户自定义域名 | `name`,`verified`,`verified_at` (UNIQUE user+name) |
| `utm_templates` | UTM 模板 | `name`,`utm_source/medium/campaign/term/content` |
| `api_tokens` | 开放 API Token | `name`,`token_hash`(UNIQUE) |
| `workspaces` | 工作区 | `name`,`slug`(UNIQUE) |

### 设计要点

- `short_code` 加了**唯一索引**（`UNIQUE`），这是短码冲突的最终仲裁。
- `click_logs` 里 `platform` 用了 **PostgreSQL ENUM**（`click_platform`），比字符串更紧凑、更约束。
- 多表都用**外键 + 级联策略**：`click_logs ON DELETE CASCADE`（删链接级联删日志）、`folders/tags ON DELETE CASCADE`（删用户级联删），`links.user_id ON DELETE SET NULL`（删用户保留链接）。
- 建了**复合索引**：`idx_links_domain_code (domain, short_code)` 加速跳转查询；`idx_sms_codes_phone (phone, created_at)` 加速验证码频控。

> 面试点：`short_code` 为什么用唯一索引而不是先 SELECT 查重？因为**并发下 check-then-insert 有竞态**，两条请求可能同时通过检查再同时插入。唯一索引让数据库在插入时裁决，代码里捕获 `23505`（唯一约束冲突）错误处理。这就是经典的"先查后插"竞态问题。

```sql
-- 短链核心表
CREATE TABLE links (
    id              BIGSERIAL PRIMARY KEY,
    short_code      VARCHAR(20) UNIQUE NOT NULL,
    original_url    TEXT NOT NULL,
    domain          VARCHAR(255) DEFAULT 'kada.link',
    password_hash   VARCHAR(255),
    expires_at      TIMESTAMPTZ,
    is_active       BOOLEAN DEFAULT TRUE,
    click_count     BIGINT DEFAULT 0,
    user_id         BIGINT REFERENCES users(id) ON DELETE SET NULL,
    workspace_id    BIGINT,
    folder_id       BIGINT,
    utm_source      VARCHAR(255),
    ios_url         TEXT,
    android_url     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 核心流程详解

### 认证：手机验证码登录

这是最容易考并发/安全题的地方。流程：

1. **发送验证码** `POST /api/auth/send-sms-code`
   - 校验手机号格式（`1[3-9]\d{9}`）。
   - **60 秒频控** + **单手机号每日 10 条上限**，防止短信轰炸/费用损失。
   - 未配置阿里云短信时，验证码只在 `GIN_MODE != release` 时打印日志（生产不落明文，超安全）。
2. **登录** `POST /api/auth/login-by-phone`
   - 用**单条原子 UPDATE** 消耗验证码：`UPDATE sms_codes SET used=TRUE WHERE id=(SELECT id FROM sms_codes WHERE phone=? AND code=? AND used=FALSE AND expires_at>NOW() AND attempts<5 ORDER BY id LIMIT 1) RETURNING id`。
   - 这个 SQL 一次完成"未使用、未过期、尝试次数未超限"三个校验，**用数据库行锁保证验证码只能被一个请求成功消费**，消除并发双用的竞态。
   - 失败时 `attempts+1`，达到 5 次该验证码作废，防爆破。
   - 登录后查/建用户（新用户自动注册），更新 `last_login_at`，签发 JWT。

**原子消耗验证码**（核心代码）：
```go
// 单条 UPDATE 同时完成「未使用、未过期、尝试次数未超限」校验，
// 消除并发请求双用同一验证码的竞态
err := s.db.QueryRow(ctx, `
    UPDATE sms_codes SET used = TRUE
    WHERE id = (
        SELECT id FROM sms_codes
        WHERE phone = $1 AND code = $2 AND used = FALSE
          AND expires_at > NOW() AND attempts < 5
        ORDER BY id
        LIMIT 1
    )
    RETURNING id
`, phone, code).Scan(&codeID)
```

> 追问：为什么用 UPDATE 而不是先 SELECT？因为 **先查后改会有竞态**，两个并发请求都能查到"未使用"的验证码，然后都消费成功。用 UPDATE 自带行锁，只有第一个事务能更新到那行 `used=TRUE`，第二个的 SELECT 子查询就找不到行，`RETURNING` 为空 → 报"验证码错误"。这是把"并发控制"下沉到数据库的经典手法。

### 短链创建

`POST /api/links`（需登录），核心逻辑在 `LinkService.Create`：

1. 校验目标 URL 协议（仅 http/https，防存储型 XSS）。
2. 校验文件夹/工作区归属当前用户（防跨用户泄漏）。
3. 生成短码：有自定义短码则校验格式（`^[a-zA-Z0-9_-]{4,20}$`）+ 查重；无则随机生成 **6 字节 → 12 位十六进制**（48bit 熵）。
4. 解析过期时间、密码 bcrypt 哈希。
5. **INSERT 捕获 `23505`**：自定义短码冲突直接报错；随机短码冲突则换码重试最多 5 次。
6. 关联标签（仅允许自己的标签）。
7. 写入 Redis 缓存。

**短码生成**（面试常问碰撞概率）：
```go
// 6字节随机 → 12位十六进制，48bit 熵
// 约 1670 万条链接才达 50% 生日碰撞概率
func generateShortCode() string {
    b := make([]byte, 6)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

> 追问：短码为什么用 12 位 hex 而不是 Base62？
> - 12 位 hex = 48 bit，`生日悖论`下约 `1.18 * sqrt(2^48) ≈ 1670万` 条时达到 50% 碰撞率。如果量级在千万级以内完全够用。
> - 实现简单、无大小写敏感歧义（`0/O`、`1/l`），复制粘贴友好。
> - Base62 可在同样长度下容纳更多（如 8 位 Base62 ≈ 47 bit），但需要自己写进制转换；权衡后 12 位 hex 是"短 + 够用 + 简单"的折中。若未来数据量大，可迁移到 Base62 或增加位数。

**唯一约束仲裁冲突**（体现工程严谨）：
```go
for attempt := 0; ; attempt++ {
    err := s.db.QueryRow(ctx, `INSERT INTO links (...) RETURNING ...`, ...).Scan(&info)
    if err == nil { break }
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "23505" {
        if 是自定义短码 { return errors.New("该短码已被占用") }
        if attempt >= 4 { return errors.New("生成短码失败") }
        shortCode = generateShortCode()  // 随机短码换码重试
        continue
    }
    return errors.New("创建短链接失败")
}
```

### 短链跳转 redirect

公开端点 `GET /r/:code`（不限登录）。整个跳转做了大量防御：

```
GET /r/:code
  │
  ├─ GetByCode（先查 Redis 缓存，miss 再查库）
  │    └─ 检查 expires_at 过期 → 返回"链接已过期"
  ├─ urlcheck.IsSafeTarget → 阻止 javascript:/data: 等协议（防 XSS）
  ├─ HasPassword → 有密码则渲染"输入密码页"
  ├─ 记录点击（LogClick，Kafka/降级直写）
  ├─ ua.Detect 平台检测
  │    ├─ 微信/QQ/小红书 → 渲染中间引导页 + QR + deeplink 尝试
  │    └─ 普通浏览器 → 302 直接跳转
  └─ 302 Redirect
```

> 面试点：为什么微信等内置浏览器需要中间页？
> 微信/QQ 内置浏览器会**屏蔽或限制外链跳转**（尤其到非白名单域名），直接 302 可能被拦截或提示"非官方网页"。中间引导页让用户明确选择"打开链接/复制链接/扫码/在浏览器打开"，并通过 JS 尝试 `intent://` deeplink 唤起浏览器，提升转化率——这是**营销短链**的核心业务价值。

### 点击事件异步链路（Kafka）

点击事件如果同步写库会拖慢跳转（每次点都 INSERT + UPDATE + 锁）。所以：

1. **Producer（API）**：`LogClick` 把 `ClickEvent` JSON 序列化后 `WriteMessages` 到 Kafka `clicks` topic。
2. **Worker（消费）**：独立进程消费，`FetchMessage`（不自动提交）→ 落库 `WriteClick` → 成功才 `CommitMessages`。

**`LogClick` 的降级策略**（体现"不丢数据"思想）：
```go
func (s *LinkService) LogClick(ctx, linkID, ip, userAgent, platform, referer) {
    if s.kafka != nil {
        if err := s.kafka.PublishClick(ctx, mq.ClickEvent{...}); err != nil {
            log.Printf("kafka publish failed, fallback to direct write: %v", err)
        } else {
            return
        }
    }
    if s.clickWriter != nil {
        s.clickWriter.WriteClick(ctx, linkID, ip, userAgent, platform, referer, time.Now())
    }
}
```

**Worker 落库 + 计数**（事务保证原子性）：
```go
func (s *ClickStore) WriteClick(ctx, linkID, ip, userAgent, platform, referer, createdAt) error {
    tx, _ := s.db.Begin(ctx)
    defer func() { _ = tx.Rollback(ctx) }()
    if _, err := tx.Exec(ctx, `INSERT INTO click_logs (...) VALUES (...)`, ...); err != nil {
        return err
    }
    if _, err := tx.Exec(ctx, `UPDATE links SET click_count = click_count + 1 WHERE id = $1`, linkID); err != nil {
        return err
    }
    return tx.Commit(ctx)
}
```

> 追问：为什么用 `FetchMessage` 而不是 `ReadMessage`？
> `ReadMessage` 会自动提交 offset，如果处理失败也提交了，**消息就永久丢失**。用 `FetchMessage` + 手动 `CommitMessages`，只有确认 `WriteClick` 成功才提交。处理失败会重试，保证"at-least-once"。

**毒消息防御**（worker 的核心细节）：
```go
// 单条消息最大重试次数，超过则提交 offset 丢弃，防止单分区消费组永久卡死
const maxDeliveryAttempts = 3

func isPermanentError(err error) bool {
    var syntaxErr *json.SyntaxError
    if errors.As(err, &syntaxErr) { return true }       // 非法 JSON，重试永远失败
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "23503" { return true } // 外键违反，如链接已删
    return false
}
```

> 追问：如果一条消息一直处理失败会发生什么？如果是**永久错误**（毒消息），Kafka 默认会无限重试，**阻塞整个分区**，后续消息全部被卡住。所以 worker 用 `attemptTracker` 在内存里记录每条消息的重试次数，达到上限或判定为永久错误就 `CommitMessages` 丢弃，确保消费组继续前进。

**`ensureTopic` 幂等创建**：
```go
// 必须在创建 reader 之前调用：若 reader 在 topic 自动创建期间加入消费组，
// kafka-go 会拿到空分配并永久卡死（segmentio/kafka-go#585）
func ensureTopic(ctx, brokerList, topic) error { ... ctrlConn.CreateTopics(...) ... }
```

> 这是踩过坑才知道的细节——体现你对 kafka-go 的深入了解。冷启动时 `healthcheck` 可能早于 controller 就绪，带退避重试 10 次。

---

## 关键机制逐个拆解

### 缓存策略（Redis）

`GetByCode` 是**读多写少**的热点路径，用 Redis 缓存短码 → `LinkInfo`。

```go
func (cs *CacheService) GetLink(ctx, shortCode) (*LinkInfo, bool) {
    data, err := cs.client.Get(ctx, cs.key("link", shortCode)).Bytes()
    if err != nil { return nil, false }     // miss
    var info domain.LinkInfo
    json.Unmarshal(data, &info)
    return &info, true
}

func (cs *CacheService) SetLink(ctx, info *LinkInfo) {
    key := cs.key("link", info.ShortCode)
    data, _ := json.Marshal(info)
    cs.client.Set(ctx, key, data, 10*time.Minute)   // TTL 10min
}
```

**缓存失效**：更新/删除链接时，先查出旧短码，然后 `InvalidateLink` 删除缓存（可能新旧两个短码都要删）。

> 面试点：缓存一致性怎么保证？
> - 采用 **Cache-Aside（旁路缓存）**：读时先查缓存，miss 再查库写缓存；写时**先更新数据库，后失效缓存**。
> - 因为跳转读的是短码，短码是稳定的，更新的是 `original_url` 等字段，所以失效短码缓存即可。
> - 缓存设置了 **TTL 兜底（10 分钟）**，即使失效逻辑漏了，过期后也会自动回源，不会永远脏。

### 限流（滑动窗口）

用 Redis **Sorted Set（ZSet）** 实现滑动窗口限流，而不是简单计数器（固定窗口有"临界突发"问题）。

```go
pipe := rl.client.Pipeline()
pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart)) // 移除窗口外旧请求
countCmd := pipe.ZCard(ctx, key)                                     // 统计窗口内请求数
pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)}) // 记录当前
pipe.Expire(ctx, key, cfg.Window*2)                                  // 过期时间
pipe.Exec(ctx)
count, _ := countCmd.Result()
if int(count) >= cfg.Limit { c.JSON(429, ...); c.Abort(); return }
```

有 3 档：
- `Strict`（认证）：1 分钟 20 次
- `Normal`（通用 API）：1 分钟 120 次
- `Redirect`（短链跳转）：1 分钟 300 次

> 追问：滑动窗口比固定窗口好在哪？
> 固定窗口在窗口边界会有"双倍突发"（如限 100/分钟，第 59 秒发 100 个、第 61 秒又发 100 个）。滑动窗口用 ZSet 记录每个请求的时间戳，**精确统计过去 1 分钟内的请求数**，避免边界穿透。

> 追问：Redis 挂了怎么办？代码里 `pipe.Exec` 失败会 fail-open（`c.Next()` 放行），等 Redis 恢复后自动生效，无需重启进程。这是"可用性优先"——限流宁可暂时放开也不让服务不可用。但要注意：在生产安全敏感接口可能需要 fail-closed，可谈平衡。

### 真实 IP 获取

后端**不信任客户端可伪造的 `X-Forwarded-For`**（它可被任意伪造，曾被用来绕过限流）。只信任 **nginx 用 `$remote_addr` 覆写的 `X-Real-IP`**。

```go
func RealIP(c *gin.Context) string {
    if ip := c.GetHeader("X-Real-IP"); ip != "" {
        if parsed := net.ParseIP(strings.TrimSpace(ip)); parsed != nil {
            return parsed.String()
        }
    }
    return c.ClientIP()
}
```

服务端还 `r.SetTrustedProxies(nil)`，让 Gin 不信任任何代理头，从源头杜绝伪造。

### 安全设计

这套项目的安全点非常密集，是**高分项**：

| 风险 | 防御 | 代码位置 |
|------|------|----------|
| JWT **算法混淆**（alg=none / RS256）| `jwt.WithValidMethods([]string{"HS256"})` + `WithExpirationRequired()` | middleware/auth.go |
| **弱密钥**生产可用 | `config.IsWeakJWTSecret`，release 模式启动即 Fatal | config.go + main.go |
| 短链目标 **XSS**（`javascript:` 协议）| `urlcheck.IsSafeTarget` 只允许 http/https | urlcheck.go |
| **SSRF**（preview 抓取内网/元数据）| 内网 CIDR 黑名单 + 仅 80/443 + 拨号时复核 + 限制重定向 | preview/fetcher.go |
| 验证码**并发双用/爆破** | 原子 UPDATE + attempts < 5 | auth_service.go |
| 短信**轰炸** | 60s 频控 + 每日 10 条 | auth_service.go |
| **CSV 公式注入** | `escapeCSV` 给 `= + - @ \t` 开头字段加 `'` 前缀 | link_service.go |
| API Token **明文泄露** | 只存 SHA-256 哈希，Token 仅返回一次 | api_token_service.go |
| 登录**邮箱枚举** | 密码错误与账号不存在返回同样文案 | auth_service.go |
| 跨用户访问 | 所有查询强制带 `user_id` 条件 | 各 service |
| 相对路径 | `SetTrustedProxies(nil)` + X-Real-IP | main.go + ratelimit.go |

**JWT 防御算法混淆**（重点背）：
```go
token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
    return []byte(secret), nil
},
    jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),  // 只允许 HS256
    jwt.WithExpirationRequired(),                                   // 必须带过期时间
)
```

> 追问：为什么加 `WithValidMethods`？历史上 JWT 有 `alg=none` 绕过漏洞，攻击者把 header 的 `alg` 改成 `none` 即可伪造 token。显式限定 HS256 并强制要求过期时间，杜绝算法混淆和无期限令牌。

**SSRF 防护**（preview 抓网页元数据，最容易被攻击）：
```go
// 拨号前再次校验（DNS rebinding 最终防线）
DialContext: func(ctx, network, addr) (net.Conn, error) {
    host, _, _ := net.SplitHostPort(addr)
    if err := validateHost(ctx, host); err != nil { return nil, err }
    return dialer.DialContext(ctx, network, addr)
}
// 仅允许 http/https 且端口 80/443 且目标 IP 不在内网/保留黑名单（169.254.169.254 云元数据等）
```

**CSV 公式注入**：
```go
func escapeCSV(s string) string {
    s = strings.ReplaceAll(s, "\r", "")
    if strings.HasPrefix(s, "=") || strings.HasPrefix(s, "+") ||
       strings.HasPrefix(s, "-") || strings.HasPrefix(s, "@") || strings.HasPrefix(s, "\t") {
        s = "'" + s   // 防止 Excel 把 =formula 当作公式执行
    }
    if strings.ContainsAny(s, ",\"\n") {
        s = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
    }
    return s
}
```

### 平台检测与中间引导页

`ua.Detect` 根据 User-Agent 字符串识别微信/QQ/微博/小红书，命中特定平台就渲染中间页。

```go
func Detect(userAgent string) domain.Platform {
    ua := strings.ToLower(userAgent)
    if strings.Contains(ua, "micromessenger") { return domain.PlatformWechat }
    // 注意顺序：先 QQ 再微信（QQ 可能也含 mqqbrowser）
    if strings.Contains(ua, "qq/") || strings.Contains(ua, "mqqbrowser") { return domain.PlatformQQ }
    if strings.Contains(ua, "xhs") || strings.Contains(ua, "redapp") { return domain.PlatformXiaohongshu }
    return domain.PlatformBrowser
}
```

中间页里用 `window.location.href`、`intent://` scheme 尝试唤起外部浏览器，并内嵌 QR 码 + 复制按钮。**注意**：`urlcheck.IsSafeTarget` 在渲染前也做了一次校验，防止存量脏数据里的 `javascript:` 协议在 `window.location.href` 中被执行（存储型 XSS 的最后防线）。

---

## 前端架构

Next.js 16 **App Router** + React 19 + TypeScript + Tailwind 4 + SWR。

```
frontend/src/
├── app/
│   ├── page.tsx              # 首页（星空背景）
│   ├── (auth)/login/         # 登录（手机/邮箱 tab）
│   ├── (auth)/register/      # 注册
│   ├── dashboard/            # 管理后台（layout 统一侧边栏）
│   │   ├── page.tsx          # 链接列表
│   │   ├── links/new/        # 创建链接
│   │   ├── links/[id]/       # 链接详情
│   │   ├── analytics/        # 分析
│   │   ├── events/           # 事件
│   │   ├── customers/        # 客户
│   │   ├── domains/ folders/ tags/ utm/ settings/
│   └── r/[code]/             # 短链跳转前端兜底页
├── components/               # AppLayout/Sidebar/LinkCard/Toolbar/Starfield...
└── lib/                      # api.ts / auth.ts / utils.ts / starfield.ts
```

**数据请求**用 SWR（`useSWR`），自带缓存、错误重试、自动刷新。搜索做了 **300ms 防抖**，创建链接时的 URL Preview 也做了 **800ms 防抖** 避免频繁请求后端。

**认证状态**存在 `localStorage`（`kada_token` / `kada_user`），`AppLayout` 里初始化时读取并做登录拦截（无 token 即跳 `/login`）。

**r/[code] 页面**：前端跳转兜底（`window.location.href = /r/${code}`），保证即使访问前端也能正常重定向。

> 面试点：为什么用 SWR 而不是自己写 fetch？SWR 提供**请求缓存、焦点重新验证、乐观更新**，减少重复请求和状态样板代码。缺点是 Token 存在 localStorage 有 XSS 风险（可谈权衡：短链接平台非密钥级敏感，属可接受）。

---

## 部署与 CI/CD

### Docker Compose 服务（docker-compose.yml）

| 服务 | 镜像/构建 | 端口 | 说明 |
|------|-----------|------|------|
| postgres | postgres:16-alpine | 127.0.0.1:5432 | 数据卷挂载迁移脚本 |
| redis | redis:7-alpine | 不发布 | 内网供 backend |
| backend | 构建 `./backend` | 8080 | API，`JWT_SECRET` 必填（`:?` 强制） |
| kafka | apache/kafka:3.8.0 | 127.0.0.1:29092 | 单节点 KRaft 模式 |
| kafka-worker | 构建 `./backend` | — | `command: ["/worker"]` 消费点击 |
| frontend | 构建 `./frontend` | 3000 | Next.js standalone |
| nginx | nginx:alpine | 80 | 反向代理，统一入口 |

**Compose 里的安全细节**：
- `JWT_SECRET: ${JWT_SECRET:?...}` 强制要求设置，不能为空。
- Postgres 密码默认可被 `.env` 覆盖（生产必须改）。
- Kafka 的 9092 只在内网，29092 才暴露给宿主机；postgres/redis 都只绑定 127.0.0.1。
- 后端 Dockerfile **以非 root 用户运行**（`USER kada`），前端用 `USER node`，体现容器安全。

### CI/CD（GitHub Actions）

**CI workflow（main 分支 push / PR）** 7 个 job：
1. `backend-lint`：golangci-lint
2. `backend-test`：起 Postgres service → `go vet` + `go test ./... -race -cover`
3. `frontend-lint`：eslint
4. `frontend-build`：`tsc --noEmit` + `next build`
5. `deploy`（仅 main push）：构建 → 备份 → **golang-migrate 跑迁移** → 替换后端 → 健康检查（失败自动回滚）→ 替换前端 → 重启 → curl 验证
6. `pipeline-summary`：汇总表

**Release workflow（打 tag v* 触发）**：跑测试 → buildx 构建 backend/frontend 镜像 → push 到 GHCR。

> 面试点：部署流程里有严谨的 **备份 + 回滚 + 健康检查**：迁移前备份旧版本，后端启动失败自动回滚旧二进制，部署后 curl `/api/health` 验证。这是工程化思维的关键体现。

> 追问：数据库迁移用哪种方式？用 `golang-migrate`（`backend/cmd/migrate`）。之前用 `psql -f` 逐个执行 up.sql，已应用迁移报错被吞掉、无状态表、可能半应用/重复执行；改用 golang-migrate 后有 `schema_migrations` 状态表，支持 `up`/`down`，可追踪版本。

---

## 测试

后端有 `_test.go`：
- `link_service_test.go`：短码格式、短码生成、CSV 转义、URL 构建、bcrypt 哈希
- `auth_service_test.go`：手机号正则校验
- `redirect/handler_test.go`、`ratelimit_test.go`、`auth_test.go`、`click_store_test.go`、`worker/main_test.go`、`urlcheck_test.go`、`preview/fetcher_test.go` 等

前端用 **Vitest**（`frontend/src/lib/*.test.ts`，如 `ophiuchus`、`starfield`）。

CI 里跑 `go test -race -coverprofile`+ `upload coverage`。**-race** 是 Go 的竞态检测，能抓并发 bug。

> 面试点：测试怎么兼顾？核心是**对纯函数和边界条件做表驱动测试**（如短码格式、CSV 转义），对依赖外部（DB/Kafka）的部分用**接口 mock**。`-race` 保证并发安全。

---

## 面试题大全（含追问）

### 第一梯队：必问基础

**Q1：介绍一下这个项目？**

答：Kada 是一个短链接 SaaS 平台。核心能力：登录注册（手机验证码/邮箱/微信）、短链的创建/管理/编辑/删除、自定义域名、访问密码、过期时间、UTM 模板、文件夹/标签/工作区分类、点击统计看板。架构上是 Go 后端（Handler→Service→Infra 三层）+ Next.js 前端，用 Docker Compose 编排 6 个容器，GitHub Actions 做 CI/CD 和自动部署。技术亮点是：Kafka 异步点击流 + 限流 + 平台检测中间页 + 一系列安全防御。

---

**Q2：短链跳转的完整流程？**

答：用户访问 `GET /r/:code`。第一步，`GetByCode` 先查 Redis 缓存，miss 再查 PostgreSQL，并校验是否过期。第二步，用 `urlcheck.IsSafeTarget` 校验目标 URL 是否 http/https，防止 `javascript:` 等协议。第三步，`HasPassword` 判断是否设了密码，有则渲染输入密码页。第四步，`LogClick` 异步记录点击事件（发 Kafka 或降级直写）。第五步，用 `ua.Detect` 检测平台：微信/QQ/小红书渲染中间引导页（带 QR、复制、deeplink 唤起），普通浏览器直接 302 跳转。

---

**Q3：短码重复怎么处理？（必考并发）**

答：短码字段有唯一索引。创建时如果同时有多个请求生成相同短码，`INSERT` 会抛 `23505` 唯一冲突。代码里捕获这个错误：自定义短码直接告诉用户"已被占用"；随机短码则重新生成，重试最多 5 次。这样即使先查后插有竞态，最终由数据库的唯一约束来裁决，保证不会出现重复短码。同时随机短码用 6 字节（48bit）熵，碰撞概率在千万量级很低。

---

**Q4：验证码登录怎么防止并发重放？**

答：用一条原子的 `UPDATE sms_codes SET used=TRUE WHERE id=(SELECT id FROM sms_codes WHERE ... AND used=FALSE AND attempts<5 ...) RETURNING id`。这条 SQL 同时完成"未使用 + 未过期 + 尝试次数未超限"校验。因为 UPDATE 会对目标行加锁，两个并发请求只有一个能更新到 `used=TRUE`，另一个的 SELECT 子查询找不到匹配行，`RETURNING` 为空，就返回验证码错误。这样彻底避免一个验证码被并发使用两次。同时尝试 5 次后作废，防止爆破。

---

**Q5：点击计数为什么用 Kafka 不用同步写？**

答：跳转是高频低延迟路径，如果每次点击都同步 `INSERT` 日志 + `UPDATE` 计数，数据库压力大、跳转变慢。用 Kafka 削峰：API 只把点击事件发到 topic（毫秒级），然后立即 302 跳转；独立 worker 进程异步消费并落库。这样 API 保持轻量，高峰点击也不会拖垮跳转。另外 Kafka 不可用时我会降级直写数据库，保证点击不丢失。

---

### 第二梯队：进阶追问

**Q6：为什么用 FetchMessage 而不是 ReadMessage？**

答：`ReadMessage` 会在收到消息后**自动提交 offset**，如果后续处理失败，offset 已经提交了，这条消息就永久丢失。用 `FetchMessage` + 手动 `CommitMessages`，只有确认落库成功才提交。这样能达到 at-least-once 语义。代价是可能重复处理（at-least-once 本来就可能重复），所以 `WriteClick` 要幂等。

---

**Q7：如果一条消息一直处理失败会怎样？**

答：会阻塞对应分区的消费进度（Kafka 默认 retry 无限次）。所以我的 worker 有 `maxDeliveryAttempts=3` 的上限，同时在内存用 `attemptTracker` 记录每条消息重试次数，够 3 次或判定为**永久错误**（非法 JSON、外键违反 23503）就直接 `CommitMessages` 丢弃，让消费组继续前进。这是防止"毒消息"让单分区单消费组永久卡死。

---

**Q8：JWT 怎么防止算法混淆攻击？**

答：我用 `jwt.ParseWithClaims` 时加了 `WithValidMethods([]string{"HS256"})` 和 `WithExpirationRequired()`。历史上 JWT 有 `alg=none` 绕过漏洞，攻击者把 header 的 alg 改成 none 就能伪造 token，验签时直接通过。显式限定只有 HS256 并强制要求过期时间，就把这类攻击堵死了。另外生产环境我还会用 `IsWeakJWTSecret` 检查弱密钥，release 模式启动就 Fatal。

---

**Q9：short_code 为什么用 hex 而不用 base62？**

答：12 位 hex 是 48bit 熵，按生日悖论大约 1670 万条链接才达到 50% 碰撞率，这个量级下够用。实现上直接用 `crypto/rand` + `hex.EncodeToString` 非常简单。Base62 能在同样长度容纳更多（8 位 ≈ 47bit），但需要自己写进制转换，还引入大小写敏感和字符混淆（0/O、1/l）的问题。权衡后选择"简单 + 够用 + 无歧义"的 12 位 hex。数据量大到千万级时再迁移 Base62 或加位数。

---

**Q10：redis 缓存一致性怎么做？**

答：采用 Cache-Aside 模式。读：先查缓存，miss 再查库并写缓存（TTL 10min）。写：更新/删除链接时，先查出旧短码，然后失效对应缓存（可能新旧两个短码都删）。因为短码是稳定的，变的只是内容字段，所以失效短码键即可。TTL 兜底保证即使失效逻辑漏了，缓存也会自动过期回源。这是"先更新数据库，后删缓存"的标准做法，避免"先删缓存再写库"导致短暂读到旧数据。

---

**Q11：限流为什么用滑动窗口？**

答：固定窗口在边界有突刺问题，比如限制 100/分钟，第 59 秒来了 100 个、第 61 秒又能来 100 个，2 秒内放行 200 个。滑动窗口用 Redis ZSet 把每个请求的时间戳记为 score，每次请求用 `ZRemRangeByScore` 移除窗口外旧记录、`ZCard` 统计窗口内数量。这样"过去 1 分钟内的请求数"是精确的，避免边界穿透。我分了 Strict/Normal/Redirect 三档，应对不同接口的流量特征。

---

**Q12：怎么防止 SSRF？**

答：`preview` 功能要抓取用户传入 URL 的网页元数据，是 SSRF 重灾区。我做了多层防护：只允许 http/https 协议、只允许 80/443 端口、DNS 解析后逐一检查目标 IP 是否在内网/保留地址黑名单（包括 169.254.169.254 云元数据、10.x、192.168.x 等）、限制最多 3 次重定向且每次重定向均校验、拨号时再复核一次（防 DNS rebinding）。这样即使攻击者传内网地址或利用 DNS 解析到内网，也会被拦截。

---

**Q13：中间引导页的意义？以及 XSS 风险？**

答：微信/QQ 内置浏览器会限制外链跳转，直接 302 可能被拦截，所以对检测到的平台渲染引导页，让用户明确选择打开/复制/扫码/唤起外部浏览器，提升转化。风险在于目标 URL 会在 `window.location.href` 里执行，如果存量脏数据是 `javascript:` 协议就可能造成存储型 XSS。所以跳转前我会用 `urlcheck.IsSafeTarget` 再校验一次协议，非法协议就渲染"链接不可用"提示页，阻止执行。

---

**Q14：为什么不用 ORM？**

答：项目里表结构复杂（links 关联 folders/tags/workspaces，点击统计要 join click_logs），而且有动态条件查询（列表页根据 search/folder/tag/workspace/sort 拼接 WHERE）。手写原生 SQL 更可控：能精确控制查询、用 COALESCE 处理空值、用 RETURNING 拿插入结果、能针对热点 SQL 加索引优化。pgx 本身就高性能，原子操作（如验证码 UPDATE）也必须有原生 SQL 才能实现。ORM（GORM）在这种多表 join + 动态查询 + 原子语句场景下反而更难写、性能和可控性都打折。

---

**Q15：为什么用 Go？**

答：短链是典型的高并发、低延迟网络服务。Go 有 goroutine + channel 的轻量并发模型，处理大量并发跳转很高效；编译型语言性能高，GC 停顿低；单二进制部署方便（+不依赖运行时）；标准库和生态（Gin、pgx、kafka-go）成熟。相比 Node/Python，Go 更适合做这种对延迟和吞吐敏感的核心链路。

---

### 第三梯队：场景/设计题（拉开差距）

**Q16：如果短链访问量突然暴涨（比如上热搜），你的系统哪里会先撑不住？怎么应对？**

答：
- 第一层压力在 **Nginx**：需要开 keepalive，调 worker_connections，必要时加 `limit_req`。
- **Redis 缓存**：`GetByCode` 先走缓存，命中率上来就能扛住大部分读。缓存可以加**热点短链的额外保护**（如热点 key 加本地缓存 / 二级缓存，防止单 key 打挂 Redis）。
- **Go API**：无状态可水平扩展，前面再挂一层负载均衡/更高级的 Nginx，多副本部署。
- **数据库**：`links` 表按 `short_code` 有唯一索引，读没问题；真正压力在 `click_logs` 写。已经用 Kafka 削峰，worker 可以**水平扩展消费组**（增加分区数 + 多个 worker 实例）。
- 前端/静态资源走 CDN。
- 最终是限流兜底（Redirect 档 300/分钟/IP），防止恶意刷量。还可以加**热点短链识别**：`click_count` 高的链接单独缓存到内存。

> 追问：Kafka 消费数据量大于落库速度怎么办？答：增加分区（从 1→N）提升并发度；worker 水平扩容；必要时对大表归档、click_logs 按时间分区。

---

**Q17：如果要支持"短链到期自动下线"，怎么设计？**

答：现在 `GetByCode` 里已经判断 `expires_at` 是否过期并返回"链接已过期"。可以再加：
- 惰性 + 定时双路径：查询时惰性判断（已有）+ 定时任务/worker 批量把过期链接 `is_active=FALSE`。
- 用 Redis 给过期链接设置 key 的 TTL 到 `expires_at`，到期自动失效缓存。
- 或者用数据库定时任务（pg_cron）批量更新。

---

**Q18：多租户隔离怎么做的？（工作区）**

答：所有关键表（links、folders、tags、domains、workspaces）都带 `user_id`，且**每个查询/更新/删除都强制带 `user_id` 条件**，防止横向越权。比如 `SELECT ... FROM links WHERE id=$1 AND user_id=$2`，改不到别人的数据。工作区 `workspace_id` 也做了归属校验（`validateOwnedRefs`）。创建链接时校验 folder/workspace 属于当前用户，标签也校验归属。这是最基础的**行级权限隔离**。

---

**Q19：如果要求强一致性（across API + worker），有多少活？**

答：目前是"最终一致"：点击事件先发 Kafka，worker 异步落库，`click_count` 可能短暂落后于实际点击。如果要求强一致，需要：
- 简化：跳转后同步更新 `click_count`（放弃异步，接受延迟）。
- 折中：读多写少场景用**计数聚合**，看板用定时汇总表。
- 需要给 `click_count` 加分布式锁（如 Redis SETNX）防并发写丢，或在数据库用 `UPDATE ... SET click_count=click_count+1` 保证原子累加。
- Kafka 会有"重复消费"（at-least-once），计数可能多算，需要幂等（如按 `message_id` 去重）。可谈"业务上点击量允许少量误差，追求吞吐"的权衡。

---

**Q20：讲讲你这个项目你最有成就感/最难的点？**

答：我会说 Kafka 点击流 + 毒消息防护那一块。因为点击事件是异步关键路径，既要保证不阻塞跳转、又要保证不丢消息、还要防止毒消息卡死整个消费组。我用了 `FetchMessage` 手动提交、`attemptTracker` 内存计数重试、`isPermanentError` 识别永久错误、`ensureTopic` 提前创建避免 kafka-go 空分配卡死。这些细节都是踩坑/读源码总结出来的，体现对消息队列可靠性的深入理解。

---

## 答不出来的兜底表达

面试被问到一个没准备的点，别慌，用这些话兜底：

1. **"这块我目前是这么理解的，可能不够全面。"** + 给出一个合理的工程直觉。
2. **"我们实际实现里是……，如果考虑到更极端的场景，我会这样改进……"** + 主动说出一个加分项（如限流 fail-open、缓存一致性、幂等）。
3. **"这个我没深入做，但我知道大概的方向是……"** + 讲原理。
4. 主动引导：**"这个功能我用了 XX 方案，可以展开讲一下它的取舍。"** 把话题拉到你的舒适区。

---

## 背熟这几个关键词（一句话答辩）

- **架构**：三层 Handler→Service→Infra，模块化单体 + 独立 worker，无 ORM 手写 SQL。
- **核心链路**：短链跳转 = 缓存查询 + 协议校验 + 密码检查 + 平台检测 + 异步点击 + 302。
- **异步**：Kafka 削峰，FetchMessage 手动提交（at-least-once），毒消息丢弃，Failed 降级直写。
- **安全**：JWT 算法混淆、弱密钥、URL 协议白名单、SSRF 黑名单、验证码原子消费、CSV 注入、行级权限。
- **并发**：短码 23505 唯一仲裁、验证码 UPDATE 行锁、点击计数原子累加。
- **性能**：Redis 短码缓存（Cache-Aside + TTL 兜底）、滑动窗口限流、Nginx 代理 + X-Real-IP。
- **工程**：golang-migrate 版本迁移、备份回滚、健康检查、Docker 非 root、CI 全链路 lint/test/build/deploy。

---

祝你面试顺利！🎯
