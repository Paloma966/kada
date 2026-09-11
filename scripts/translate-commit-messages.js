#!/usr/bin/env node
/*
 * Rewrite the remaining Chinese commit subjects and bodies in this repository to
 * English.
 *
 * The translation map is self-contained, so this needs nothing but Node and git.
 * Run it from the repository root:
 *
 *     node scripts/translate-commit-messages.js
 *
 * It rewrites refs/heads/main in place. filter-branch keeps the previous history at
 * refs/original/refs/heads/main until you remove it, so nothing is lost:
 *
 *     git update-ref -d refs/original/refs/heads/main   # only after verifying
 *
 * Nothing has been pushed, so publishing needs
 * `git push --force-with-lease origin main`.
 */
const { execFileSync } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const SUBJECTS = {
  "fix: 阻止 javascript:/data: 协议短链，修复引导页存储型 XSS": "fix: block javascript: and data: short links, fix stored XSS on guide page",
  "fix: 修复 preview 端点 SSRF（内网/云元数据可达）": "fix: SSRF in preview endpoint (internal network and cloud metadata reachable)",
  "fix: 修复伪造 X-Forwarded-For 绕过限流与 IP 伪造": "fix: prevent forged X-Forwarded-For from bypassing rate limiting and spoofing IPs",
  "fix: 短信验证码不再明文写入日志": "fix: stop logging SMS codes in plaintext",
  "fix: 短链访问密码改用 bcrypt 哈希": "fix: hash short link access passwords with bcrypt",
  "fix: 短信轰炸防护（格式校验+冷却+日上限）": "fix: protect against SMS bombing (format validation, cooldown, daily cap)",
  "fix: 短信验证码并发竞态与单码爆破防护": "fix: prevent SMS code race conditions and single-code brute force",
  "fix: Kafka worker 毒消息不再永久卡死消费组": "fix: stop Kafka worker poison messages from stalling the consumer group forever",
  "fix: CI 部署迁移改用 golang-migrate，消除半应用状态": "fix: use golang-migrate in CI deploy to avoid half-applied migrations",
  "fix: 域名所有权验证改为真实 DNS TXT 校验": "fix: verify domain ownership with a real DNS TXT lookup",
  "fix: 生产环境拒绝弱 JWT 密钥，compose 强制显式配置": "fix: reject weak JWT secrets in production and require explicit config in compose",
  "fix: 二维码生成使用链接实际域名": "fix: use the link actual domain when generating QR codes",
  "fix: UpdateUser 空字段时返回零值用户结构体": "fix: return zero-value user when UpdateUser fields are empty",
  "fix: UpdateMe 校验邮箱格式 + 登录错误信息防邮箱枚举": "fix: validate email format in UpdateMe and prevent email enumeration on login",
  "fix: CSV 导出字段转义与公式注入防护": "fix: escape CSV export fields and prevent formula injection",
  "fix: 随机短码熵 32bit→48bit + 唯一冲突竞态处理": "fix: bump random short code entropy from 32 to 48 bits and handle unique collisions",
  "fix: 非法过期时间返回错误而非静默忽略": "fix: return an error for invalid expiry instead of silently ignoring it",
  "fix: 新窗口打开的外部链接加 rel=noopener noreferrer": "fix: add rel=noopener noreferrer to external links opened in new tabs",
  "fix: 内部错误信息不再泄漏给客户端": "fix: stop leaking internal errors to clients",
  "fix: Redis 故障恢复后限流自动恢复，无需重启": "fix: restore rate limiting after Redis failure without restart",
  "fix: JWT 中间件显式限定 HS256 并要求过期时间": "fix: restrict JWT middleware to HS256 with required expiry",
  "fix: nginx 补充安全响应头": "fix: add security response headers in nginx",
  "fix: 容器非 root 运行 + systemd 服务安全加固": "fix: run containers as non-root and harden systemd services",
  "fix: CI 安全门禁（gosec + 前端 lint/tsc 真正生效）": "fix: make CI security gates effective (gosec + frontend lint/tsc)",
  "fix: docker-compose 数据库密码改为环境变量注入": "fix: inject database password via environment variable in docker-compose",
  "fix: 移除 Makefile/setup.sh 硬编码服务器 IP": "fix: remove hardcoded server IP from Makefile and setup.sh",
  "chore: 移除未使用的 sqlc 配置与查询文件": "chore: remove unused sqlc config and query files",
  "fix: 标签/文件夹/工作区归属校验（防跨用户挂接泄漏）": "fix: validate tag/folder/workspace ownership to prevent cross-user attachment leaks",
  "fix: 面板中目标链接 href 增加协议白名单过滤": "fix: allowlist protocols for target link href in dashboard",
  "fix: 消除 govet shadow 告警（tag 归属校验变量遮蔽）": "fix: silence govet shadow warnings (shadowed tag ownership variable)",
  "docs: 添加 README 与面试讲解文档": "docs: add README and interview prep doc",
  "fix: 邮箱唯一约束改为大小写不敏感": "fix: make email unique constraint case-insensitive",
  "fix: 统一 docker-compose 环境变量名与 config 一致": "fix: align docker-compose env var names with config",
  "fix: 批量删除链接后失效缓存": "fix: invalidate cache after bulk link deletion",
  "fix: 短信签名/模板改为从配置读取": "fix: read SMS sign name and template from config",
  "fix: 邮箱登录增加 dummy bcrypt 比较，防时间侧信道枚举": "fix: dummy bcrypt compare on email login to prevent timing-based enumeration",
  "feat: API 增加优雅停机": "feat: graceful shutdown for API",
  "fix: 点击事件增加幂等键，避免 Kafka 重投重复计数": "fix: idempotency key for click events to avoid duplicate counting on Kafka redelivery",
  "fix: 验证码哈希存储，并修复尝试次数防爆破失效": "fix: hash SMS codes and fix broken attempt limit",
  "docs: 移除面试讲解文档（非项目内容）": "docs: remove interview prep doc (not part of the project)",
  "docs: README 精简并移除表情": "docs: streamline README and drop emoji",
  "fix: 修复 golangci-lint 报错": "fix: resolve golangci-lint errors"
};
const BODIES = {
  "- link_service.go: BatchDelete 内层 scan err 改名 scanErr，去除 govet shadow 遮蔽": "- link_service.go: rename the inner scan err in BatchDelete to scanErr, removing the govet shadow",
  "- main.go: http.Server 增加 ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout，修复 gosec G112 Slowloris": "- main.go: add ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout to http.Server, fixing gosec G112 Slowloris",
  "- 移除原有表情符号与冗长章节": "- Remove the existing emoji and lengthy sections",
  "- 保留核心信息：功能、技术栈、架构、快速开始、环境变量、部署、目录": "- Keep the core information: features, tech stack, architecture, quick start, environment variables, deployment, layout",
  "- 迁移 000013：sms_codes 用 code_hash 替代明文 code（sha256 回填后删列）": "- Migration 000013: replace the plaintext code in sms_codes with code_hash (sha256 backfill, then drop the column)",
  "- 发送/校验均改为哈希比对，泄库也无法直接复用验证码": "- Both sending and verification now compare hashes, so a database leak cannot directly reuse verification codes",
  "- 修复原 attempts 按 code 定位导致错误验证码永不计数的问题：": "- Fix the previous problem where attempts matched by code never counted wrong verification codes:",
  "  改为按手机号累加未使用/未过期验证码的尝试次数，连续 5 次错误即作废": "  attempts now accumulate per phone number across unused/unexpired codes, invalidating after 5 consecutive failures",
  "- ClickEvent 增加 event_id，生产端生成随机幂等键": "- Add event_id to ClickEvent, with the producer generating a random idempotency key",
  "- click_logs 增加 event_id 唯一索引（迁移 000012）": "- Add a unique index on event_id in click_logs (migration 000012)",
  "- WriteClick 使用 ON CONFLICT(event_id) DO NOTHING：重复事件不再累加 click_count": "- WriteClick uses ON CONFLICT(event_id) DO NOTHING: duplicate events no longer accumulate click_count",
  "- 同步更新 worker 与相关单测": "- Update the worker and related unit tests accordingly",
  "- 改用 http.Server + signal.NotifyContext + Shutdown": "- Switch to http.Server + signal.NotifyContext + Shutdown",
  "- 监听 SIGINT/SIGTERM，给在途请求最多 10 秒完成": "- Listen for SIGINT/SIGTERM and give in-flight requests up to 10 seconds to finish",
  "- 与 worker 的优雅退出保持一致，避免部署时 systemctl stop 掐断连接": "- Stay consistent with the worker's graceful shutdown so systemctl stop does not cut connections during deploy",
  "- 账号不存在、未设密码时也执行一次 bcrypt CompareHashAndPassword": "- Also run a bcrypt CompareHashAndPassword when the account does not exist or has no password",
  "- 抹平与密码错误路径的响应耗时差异，避免通过耗时枚举已注册邮箱": "- Even out the response time against the wrong-password path, preventing timing-based enumeration of registered emails",
  "- NewAliyunSender 增加 signName/templateCode 参数": "- Add signName/templateCode parameters to NewAliyunSender",
  "- 发送验证码时使用配置值，未配置时回退默认（恒创联众/100001）": "- Use configured values when sending codes, falling back to the defaults (Hengchuang Lianzhong/100001)",
  "- 消除此前 SMSSignName/SMSTemplateCode 配置形同虚设的问题": "- Eliminate the previous problem where the SMSSignName/SMSTemplateCode config was effectively useless",
  "- 删除前先查询目标短码，删除后逐个 InvalidateLink": "- Query the target short codes before deletion, then InvalidateLink each one afterwards",
  "- 与单条 Delete 行为对齐，避免已删除链接在缓存 TTL 内仍可被跳转": "- Align with single Delete behaviour so deleted links cannot still be redirected within the cache TTL",
  "- JWT_EXPIRES -> JWT_EXPIRES_IN（config 读取的是后者）": "- JWT_EXPIRES -> JWT_EXPIRES_IN (config reads the latter)",
  "- BASE_URL -> API_BASE_URL（config 读取的是后者）": "- BASE_URL -> API_BASE_URL (config reads the latter)",
  "- 修复此前 compose 配置被静默忽略、JWT 有效期始终为默认 720h 的问题": "- Fix the previous problem where the compose config was silently ignored and the JWT lifetime stayed at the default 720h",
  "- 新增迁移 000011：对 users.email 建立 LOWER(email) 唯一索引": "- Add migration 000011: create a unique index on LOWER(email) for users.email",
  "- 应用层 normalizeEmail 统一小写存储/查询（注册、登录、更新）": "- normalise email casing in the application layer for storage and lookup (register, login, update)",
  "- 避免 User@x.com 与 user@x.com 被当作两个账号": "- Prevent User@x.com and user@x.com from being treated as two accounts",
  "- 新增 GitHub 首页 README（项目介绍、功能特性、架构、快速开始、API 概览、部署）": "- Add the GitHub homepage README (project intro, features, architecture, quick start, API overview, deployment)",
  "- 新增 docs/interview-kada.md（项目详解与面试题复习手册）": "- Add docs/interview-kada.md (detailed project walkthrough and interview prep handbook)",
  "- 新增 safeHref：仅 http/https 链接用于 <a href>，其余返回 #": "- Add safeHref: only http/https links are used for <a href>, anything else returns #",
  "- 应用于事件列表、链接详情、创建页的目标 URL 展示": "- Applied to target URL rendering in the events list, link detail and create pages",
  "- 后端写入时已拒绝 javascript:/data:，此处兜底存量数据，": "- The backend already rejects javascript:/data: on write; this backstops legacy data,",
  "  防止在面板源内点击执行脚本（纵深防御）": "  preventing script execution from clicks within the dashboard origin (defense in depth)",
  "- AddTagToLink/BatchTag 校验标签属于当前用户": "- AddTagToLink/BatchTag verify that the tag belongs to the current user",
  "- 创建/更新链接时校验 folder_id/workspace_id 归属，返回友好错误": "- Validate folder_id/workspace_id ownership when creating/updating links, returning a friendly error",
  "- 创建/更新时批量挂标签仅接受自己的标签，非归属标签跳过并记录日志": "- On create/update, bulk tagging accepts only the user's own tags; non-owned tags are skipped and logged",
  "- 此前可挂接他人标签 ID，泄漏其标签名/颜色等数据": "- Previously another user's tag ID could be attached, leaking data such as its tag name/colour",
  "- db/queries/*.sql 与 sqlc.yaml 从未被引用，服务层全部使用手写参数化 SQL，": "- db/queries/*.sql and sqlc.yaml were never referenced; the service layer uses hand-written parameterised SQL throughout,",
  "  保留会造成维护者误以为查询定义在使用": "  so keeping them would mislead maintainers into thinking the query definitions are in use",
  "- Makefile 移除 sqlc-gen 目标与 sqlc 安装步骤": "- Remove the sqlc-gen target and the sqlc install step from the Makefile",
  "- Makefile 部署目标改为 DEPLOY_HOST 变量，未设置时 check-host 明确报错": "- The Makefile deploy target now uses a DEPLOY_HOST variable, with check-host reporting a clear error when unset",
  "- setup.sh 同样改为变量占位，仓库公开时不再暴露运维入口地址": "- setup.sh likewise uses a variable placeholder, so publishing the repo no longer exposes the ops entry address",
  "- POSTGRES_PASSWORD 不再硬编码 kada123，通过 .env/POSTGRES_PASSWORD 覆盖": "- POSTGRES_PASSWORD is no longer hardcoded to kada123 and is overridden via .env/POSTGRES_PASSWORD",
  "  （默认值仅保留给本地开发）": "  (the default is kept only for local development)",
  "- backend/kafka-worker 的 DATABASE_URL 同步使用同一变量": "- DATABASE_URL for backend/kafka-worker uses the same variable",
  "- 配合 .env.example 与 JWT_SECRET 必填项，生产误部署风险收敛": "- Together with .env.example and the required JWT_SECRET, this narrows the risk of a production misdeployment",
  "- golangci-lint 启用 gosec 并修复全部发现：未处理错误统一检查/日志、": "- Enable gosec in golangci-lint and fix every finding: unchecked errors are uniformly checked/logged,",
  "  日志注入误报加 nosec 说明、开发默认连接串标注": "  log-injection false positives are annotated with nosec, and the dev default connection string is flagged",
  "- 修复前端 6 处 react-hooks/set-state-in-effect 违规（React 19 新规则）：": "- Fix 6 frontend react-hooks/set-state-in-effect violations (new React 19 rule):",
  "  AppLayout/settings 改为初始化即取本地状态，links/new 预览清空移到输入": "  AppLayout/settings now read local state at initialisation, and the links/new preview reset moved to the input",
  "  事件，链接详情页 loading 用初始 true + 异步回调更新、QR 关闭时清理": "  event, while the link detail page uses initial true for loading with async callback updates, cleaning up when QR closes",
  "- CI：ESLint 改为 npm run lint 不再 || true 吞错；tsc --noEmit 真正阻断": "- CI: ESLint now runs npm run lint without || true swallowing errors; tsc --noEmit genuinely blocks",
  "  类型错误；Makefile lint-fe-ci 同步": "  type errors; Makefile lint-fe-ci updated to match",
  "- backend 镜像新增 kada 非特权用户并以 USER 运行": "- The backend image adds a non-privileged kada user and runs as USER",
  "- frontend 镜像以 node 用户（uid 1000）运行，文件 chown 到位": "- The frontend image runs as the node user (uid 1000), with files chowned accordingly",
  "- kada-frontend.service 增加 NoNewPrivileges/ProtectSystem/ProtectHome/": "- kada-frontend.service gains NoNewPrivileges/ProtectSystem/ProtectHome/",
  "  PrivateTmp/CapabilityBoundingSet 等沙箱配置，限制被攻破后的横向能力": "  PrivateTmp/CapabilityBoundingSet and other sandbox settings, limiting lateral movement after a compromise",
  "- 生产：HSTS、X-Content-Type-Options nosniff、X-Frame-Options DENY、": "- Production: HSTS, X-Content-Type-Options nosniff, X-Frame-Options DENY,",
  "- /_next/ location 单独保留 nosniff（add_header 继承规则）": "- The /_next/ location keeps nosniff separately (add_header inheritance rules)",
  "- 开发 compose 的 nginx 同步加基础头": "- The dev compose nginx gets the same basic headers",
  "- 说明：站点 CSP 未在 nginx 层添加——Next.js 需要 unsafe-inline，": "- Note: the site CSP is not added at the nginx layer, because Next.js needs unsafe-inline,",
  "  会削弱 CSP 价值；XSS 根因已在后端协议校验层修复": "  which would weaken the value of CSP; the XSS root cause was fixed in the backend protocol validation layer",
  "- 原实现不限制签名算法，依赖密钥类型隐式防混淆；": "- The original implementation did not restrict the signing algorithm, relying implicitly on the key type to prevent confusion;",
  "  现在 WithValidMethods(HS256) + WithExpirationRequired 显式加固，": "  it is now hardened explicitly with WithValidMethods(HS256) + WithExpirationRequired,",
  "  防御 alg=none/RS256 算法混淆与无 exp 令牌": "  defending against alg=none/RS256 algorithm confusion and tokens without exp",
  "- 此前启动时 ping 失败即丢弃客户端，限流器永久为 nil，": "- Previously a failed ping at startup discarded the client, leaving the rate limiter nil forever,",
  "  Redis 恢复后服务仍无限流直到重启": "  so the service had no rate limiting even after Redis recovered, until it was restarted",
  "- NewRedis 现在 ping 失败也返回客户端（go-redis 自动重连）：": "- NewRedis now returns the client even when ping fails (go-redis reconnects automatically):",
  "  Redis 恢复前限流 fail-open，恢复后自动生效": "  rate limiting fails open until Redis recovers, then takes effect automatically",
  "- 短信发送/存储、注册、创建用户、创建 UTM 模板等路径此前把底层": "- SMS send/storage, registration, user creation, UTM template creation and similar paths previously returned underlying",
  "  DB/短信供应商错误（含约束名等内部细节）直接返回给客户端": "  DB/SMS provider errors (including internal details such as constraint names) directly to the client",
  "- 统一改为：真实错误记入日志，响应返回用户友好的通用提示": "- Unified approach: the real error is written to the log and the response returns a user-friendly generic message",
  "- 链接详情不存在时返回固定文案，不再透出英文内部错误": "- A missing link detail now returns a fixed message instead of exposing English internal errors",
  "- 5 处 target=_blank 链接指向用户可控 URL，缺少 rel 属性时被打开页面": "- Five target=_blank links pointed at user-controlled URLs; without a rel attribute the opened page",
  "  可通过 window.opener 将当前页重定向到钓鱼页（reverse tabnabbing）": "  could redirect the current page to a phishing page via window.opener (reverse tabnabbing)",
  "- 统一补充 rel=\"noopener noreferrer\"": "- Add rel=\"noopener noreferrer\" consistently",
  "- Create/Update 对无法解析的 expires_at 此前静默当作无过期时间，": "- Create/Update previously treated an unparseable expires_at silently as no expiry,",
  "  用户以为已设置过期实际永不过期": "  so users believed an expiry was set while links never expired",
  "- 现在返回明确错误提示 RFC3339 格式；空字符串视为未设置，保持兼容": "- Now a clear error states the RFC3339 format; an empty string counts as unset, preserving compatibility",
  "- 短码从 4 字节（8 位 hex，约 7.7 万条即 50% 生日碰撞）改为": "- Change the short code from 4 bytes (8 hex chars, 50% birthday collision at about 77k links) to",
  "  6 字节（12 位 hex，约 1670 万条才达碰撞线）": "  6 bytes (12 hex chars, reaching the collision line only at about 16.7 million links)",
  "- INSERT 唯一冲突（23505）作为最终仲裁：自定义短码返回友好错误，": "- Treat the INSERT unique violation (23505) as the final arbiter: custom short codes return a friendly error,",
  "  随机短码自动换码重试，消除检查-插入竞态": "  random short codes are regenerated and retried, eliminating the check-then-insert race",
  "- 创建失败不再把数据库内部错误返回给客户端": "- Creation failures no longer return internal database errors to the client",
  "- code/domain 字段此前未转义，domain 为用户可控，含逗号/引号会破坏 CSV 结构": "- The code/domain fields were previously unescaped; domain is user-controlled, and commas/quotes would break the CSV structure",
  "- escapeCSV 对 = + - @ 制表符开头字段加单引号前缀，防 Excel/DDE 公式注入；": "- escapeCSV prefixes fields starting with = + - @ or tab with a single quote to prevent Excel/DDE formula injection;",
  "  引号包裹逻辑覆盖所有字段": "  quote-wrapping now covers every field",
  "- 补充公式注入与特殊字符的测试用例": "- Add test cases for formula injection and special characters",
  "- UpdateMe 的 email 字段增加 omitempty,email 校验，防止任意字符串写入": "- Add omitempty,email validation to the UpdateMe email field so arbitrary strings cannot be written",
  "- LoginByEmail 对「账号不存在」与「未设置密码」返回相同错误信息，": "- LoginByEmail returns the same error message for \"account does not exist\" and \"password not set\",",
  "  消除通过错误文案枚举已注册邮箱": "  eliminating enumeration of registered emails through error wording",
  "- 两个字段均为空串时不执行 UPDATE 且不进入回退分支，导致响应返回 id=0 的": "- When both fields were empty strings, no UPDATE ran and the fallback branch was skipped, so the response returned an id=0",
  "  零值 UserInfo，前端资料被清空显示": "  zero-value UserInfo, which cleared the profile display in the frontend",
  "- 重构为单条 COALESCE UPDATE，无可更新内容时返回当前用户；": "- Refactor to a single COALESCE UPDATE that returns the current user when there is nothing to update;",
  "  数据库错误记录日志、响应不再泄漏内部细节": "  database errors are logged and responses no longer leak internal details",
  "此前 BuildShortURL(\"\", code) 会生成 https:///r/CODE 的无效 URL，": "Previously BuildShortURL(\"\", code) produced an invalid https:///r/CODE URL,",
  "扫码无法打开；改为使用链接的 domain 字段": "which scanning could not open; it now uses the link's domain field",
  "- config 新增 IsWeakJWTSecret：识别默认值/changeme 等已知弱密钥": "- Add IsWeakJWTSecret to config: detects known weak secrets such as the default value/changeme",
  "- server release 模式启动时若使用弱密钥直接 Fatal，防止任何人伪造登录令牌": "- The server now exits fatally at startup in release mode when a weak secret is used, preventing anyone from forging login tokens",
  "- docker-compose 的 JWT_SECRET 改为 :? 必填，缺失时 fail-closed 并提示生成命令": "- JWT_SECRET in docker-compose is now required via :?, failing closed when missing and suggesting a generation command",
  "- 新增 .env.example 说明环境变量": "- Add .env.example documenting the environment variables",
  "- 之前 Verify 直接置 verified=TRUE（假验证），任何人可声明任意域名": "- Previously Verify just set verified=TRUE (fake verification), letting anyone claim any domain",
  "- 现在要求域名 DNS 中存在 TXT 记录 kada-verify=<sha256(userID:domainID:name)前16位>": "- Now a TXT record kada-verify=<first 16 chars of sha256(userID:domainID:name)> must exist in the domain's DNS",
  "- 未验证域名在创建/列表响应中返回 verification_code 供用户配置": "- Unverified domains return verification_code in create/list responses for users to configure",
  "- 校验失败返回 400 及需要配置的 TXT 值，补充验证码确定性测试": "- Verification failure returns 400 plus the TXT value to configure, with determinism tests for the verification code",
  "- 之前每次部署用 psql -f 重放全部 up.sql：已应用迁移报错被 || echo 吞掉、": "- Previously every deploy replayed all up.sql with psql -f: errors from already-applied migrations were swallowed by || echo,",
  "  不使用 schema_migrations 状态表，迁移可能半应用或与本地状态不同步": "  and the schema_migrations state table was not used, so migrations could be half-applied or out of sync with local state",
  "- 现在额外构建 bin/migrate，部署机上执行 backend/cmd/migrate（golang-migrate），": "- Now bin/migrate is built additionally and backend/cmd/migrate (golang-migrate) runs on the deploy host,",
  "  与本地/容器内迁移机制完全一致；DATABASE_URL 通过 GitHub secret 传入": "  exactly matching the local/container migration mechanism; DATABASE_URL is passed in via a GitHub secret",
  "- 区分永久性错误（非法 JSON、外键违反 23503）与可重试错误": "- Distinguish permanent errors (invalid JSON, foreign key violation 23503) from retryable errors",
  "- 内存 attemptTracker 按 partition+offset 计数，重试上限 3 次": "- The in-memory attemptTracker counts by partition+offset, with a retry limit of 3",
  "- 永久性错误或重试超限时提交 offset 跳过并告警日志，": "- On a permanent error or exhausted retries, commit the offset to skip the message and log an alert,",
  "  防止单条毒消息让单分区消费组永久停滞": "  preventing a single poison message from stalling a single-partition consumer group forever",
  "- 可重试错误退避 1 秒继续重试，点击不丢": "- Retryable errors back off 1 second and are retried, so no click is lost",
  "- 补充 isPermanentError/attemptTracker 单元测试": "- Add unit tests for isPermanentError/attemptTracker",
  "- 新增迁移 000010：sms_codes 增加 attempts 字段": "- Add migration 000010: add the attempts field to sms_codes",
  "- 校验+置位合并为单条 UPDATE...RETURNING，原子消耗验证码，": "- Merge verification and marking into a single UPDATE...RETURNING to consume the code atomically,",
  "  消除并发请求同码双用的竞态": "  eliminating the race where concurrent requests use the same code twice",
  "- 错误尝试计数，单码最多 5 次错误尝试后作废，配合 60s 冷却防爆破": "- Count failed attempts and invalidate a code after at most 5 failures, with the 60s cooldown to prevent brute force",
  "- 手机号改为正则校验（1[3-9]+9位数字），拒绝任意 11 字符": "- Validate phone numbers with a regex (1[3-9]+9 digits) and reject arbitrary 11-character strings",
  "- 每手机号 60 秒发送冷却": "- 60-second send cooldown per phone number",
  "- 每手机号每日最多 10 条，防止批量轰炸造成短信费用损失": "- At most 10 messages per phone number per day to prevent SMS cost losses from bulk bombing",
  "- 错误信息用户友好，补充 phonePattern 单元测试": "- User-friendly error messages, plus unit tests for phonePattern",
  "- 新密码使用 bcrypt（加盐、防彩虹表），超 72 字节自动截断": "- New passwords use bcrypt (salted, rainbow-table resistant) and are auto-truncated beyond 72 bytes",
  "- 校验兼容存量未加盐 SHA-256 十六进制哈希（常数时间比较），": "- Verification stays compatible with legacy unsalted SHA-256 hex hashes (constant-time comparison),",
  "  旧数据升级后仍可访问，新数据全部走 bcrypt": "  so old data remains accessible after upgrade while all new data uses bcrypt",
  "- 更新单元测试：bcrypt 加盐唯一性、超长密码、存量哈希兼容": "- Update unit tests: bcrypt salt uniqueness, over-long passwords, legacy hash compatibility",
  "- 阿里云发送日志只记录脱敏手机号（138****1234），删除验证码": "- Alibaba Cloud send logs record only the masked phone number (138****1234); the verification code is removed",
  "- DEV 模式的验证码打印增加 GIN_MODE != release 门控，": "- Gate DEV-mode verification code printing behind GIN_MODE != release,",
  "  生产环境（含未配置短信服务的误部署）日志绝不落验证码": "  so production logs (including misdeployments without SMS configured) never contain verification codes",
  "- SetTrustedProxies(nil)：不再信任任何代理传入的 XFF": "- SetTrustedProxies(nil): no longer trust XFF from any proxy",
  "- 新增 middleware.RealIP：仅接受 nginx 覆写的 X-Real-IP（客户端无法伪造），": "- Add middleware.RealIP: only accept the X-Real-IP set by nginx (clients cannot forge it),",
  "  直连时回退 RemoteAddr；限流 key 与点击日志统一使用": "  falling back to RemoteAddr on direct connections, and used consistently for the rate limit key and click logs",
  "- 补充 RealIP 偏好/防伪造/非法值回退的单元测试": "- Add unit tests for RealIP preference, anti-spoofing and fallback on invalid values",
  "- 目标 URL 仅允许 http/https 且端口限 80/443": "- Target URLs may only use http/https, restricted to ports 80/443",
  "- 内网/保留 IP 地址段黑名单（含 169.254.169.254 云元数据）": "- Blocklist of internal/reserved IP ranges (including the 169.254.169.254 cloud metadata address)",
  "- 自定义 DialContext 拨号时再次校验，防 DNS rebinding": "- Validate again in the custom DialContext dialer to prevent DNS rebinding",
  "- 重定向逐跳校验，防止经 302 跳到内网": "- Validate every redirect hop to prevent a 302 from reaching the internal network",
  "- 补充 validateTarget/isBlockedIP 单元测试": "- Add unit tests for validateTarget/isBlockedIP",
  "- 新增 urlcheck.IsSafeTarget：目标 URL 仅允许 http/https": "- Add urlcheck.IsSafeTarget: target URLs may only use http/https",
  "- 创建/更新短链时校验协议，拒绝不安全目标": "- Validate the protocol when creating/updating short links and reject unsafe targets",
  "- 跳转与密码验证路径再次校验，防御存量脏数据": "- Validate again in the redirect and password verification paths to defend against legacy dirty data",
  "- 引导页/跳转被拦截时返回提示页，不再执行目标 URL": "- Return a notice page when the guide page/redirect is blocked instead of following the target URL",
  "Disable the 获取验证码 button while sendSMSCode is in flight (sending": "Disable the \"Get verification code\" button while sendSMSCode is in flight (sending",
  "- canvas 是替换元素，absolute inset-0 不拉伸，补 h-full w-full 铺满视口": "- canvas is a replaced element, so absolute inset-0 does not stretch it; add h-full w-full to fill the viewport",
  "- 星座 scale 30->45（占屏约 72%）": "- Constellation scale 30->45 (about 72% of the screen)",
  "- 星星远层 2000->1500、近层 320->240": "- Stars: far layer 2000->1500, near layer 320->240",
  "- 每颗星随机亮度 0.35~1.0，加变化测试": "- Random brightness 0.35~1.0 per star, with variation tests"
};

// Built with String.raw so that the `\n` sequences below reach the filter file as
// the two characters JavaScript interprets as a newline. A plain template literal
// would turn them into real newlines here and the filter would emit a literal
// backslash-n, gluing every commit body onto its subject line.
const filterSource = String.raw`
const SUBJECTS = __INJECT_SUBJECTS__;
const BODIES = __INJECT_BODIES__;
const subjectMap = new Map(Object.entries(SUBJECTS));
const bodyMap = new Map(Object.entries(BODIES));
const chunks = [];
process.stdin.on("data", (c) => chunks.push(c));
process.stdin.on("end", () => {
  const message = Buffer.concat(chunks).toString("utf8");
  const nl = message.indexOf("\n");
  if (nl === -1) {
    process.stdout.write(subjectMap.get(message) ?? message);
    return;
  }
  const subject = message.slice(0, nl);
  const head = (subjectMap.get(subject) ?? subject) + "\n";
  const body = message.slice(nl + 1).split("\n").map((line) => {
    const key = line.replace(/\s+$/, "");
    const hit = bodyMap.get(key);
    return hit === undefined ? line : hit + line.slice(key.length);
  });
  process.stdout.write(head + body.join("\n"));
});
`;

const filterFileSource = filterSource
  .replace("__INJECT_SUBJECTS__", () => JSON.stringify(SUBJECTS))
  .replace("__INJECT_BODIES__", () => JSON.stringify(BODIES));

const repoRoot = execFileSync("git", ["rev-parse", "--show-toplevel"], { encoding: "utf8" }).trim();
const filterPath = path.join(os.tmpdir(), "kada-msgfilter.js");
fs.writeFileSync(filterPath, filterFileSource, "utf8");

// Run the filter with this same node binary, quoted for the shell git uses.
const nodePath = process.execPath.replace(/\\/g, "/");
const filterCmd = `"${nodePath}" "${filterPath.replace(/\\/g, "/")}"`;

console.log("subject rules:", Object.keys(SUBJECTS).length);
console.log("body rules:", Object.keys(BODIES).length);
console.log("filter:", filterPath);
console.log("rewriting...");

try {
  execFileSync("git", ["filter-branch", "-f", "--msg-filter", filterCmd, "HEAD"], {
    cwd: repoRoot,
    stdio: "inherit",
    env: { ...process.env, FILTER_BRANCH_SQUELCH_WARNING: "1" },
  });
} finally {
  fs.rmSync(filterPath, { force: true });
}

const log = execFileSync("git", ["log", "--all", "--pretty=format:%s%x1f%b"], {
  encoding: "utf8",
  maxBuffer: 64 * 1024 * 1024,
});
const hanLines = log.split("\n").filter((l) => /[\u4e00-\u9fff]/.test(l));
console.log("");
console.log("message lines still containing Chinese:", hanLines.length);
if (hanLines.length) {
  for (const l of hanLines.slice(0, 10)) console.log("  " + l.trim().slice(0, 120));
  process.exitCode = 1;
} else {
  console.log("all commit messages are now English");
  console.log("");
  console.log("next: review with `git log --oneline`, then publish with");
  console.log("      git push --force-with-lease origin main");
}
