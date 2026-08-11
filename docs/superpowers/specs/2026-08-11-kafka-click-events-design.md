# Kafka 点击事件流 设计

日期：2026-08-11
状态：已与用户确认

## 目标

把短链点击日志从请求路径解耦成 Kafka 异步消息流：

- 重定向处理器不再直接写库，而是把点击事件发布到 Kafka
- 独立 worker 消费消息，写 `click_logs` + 更新 `click_count`
- Kafka 不可用时自动回退直写，保证点击不丢、重定向路径不受影响

这是简历项目的核心亮点：**「点击日志异步化 + 生产/消费解耦 + 优雅降级」**。

## 现状（已核实）

- `backend/internal/handler/redirect/handler.go`：`Redirect` 里 `go h.svc.LogClick(...)`（goroutine 直写）
- `backend/internal/service/link_service.go:486` `LogClick`：`INSERT INTO click_logs` + `UPDATE links SET click_count = click_count + 1`
- `LinkService` 接口（redirect handler 内）含 `LogClick`，签名保持即可零改动 handler
- `config/config.go`：`Config` 结构 + `getEnv`；新增字段走同模式
- docker-compose：postgres / redis / backend / frontend / nginx；backend Dockerfile 只构建 `/server`
- 数据层仍用 pgx（**本设计不引入 GORM / gRPC**）

## 架构

```
重定向请求
   └─ LinkService.LogClick（生产）
        ├─ 发布 click.event → Kafka topic "clicks"      （正常）
        └─ Kafka 不可用 → 回退直写 click_logs            （降级，不丢）
                    ▼
        Kafka broker（docker 单节点 KRaft）
                    ▼
        cmd/worker（独立消费者进程）
        └─ 消费 → 事务写 click_logs + 更新 click_count
```

## 范围

**新增**
- `backend/internal/mq/click.go` — ClickEvent + ClickPublisher 接口 + Kafka 实现
- `backend/internal/service/click_store.go` — ClickStore（直写逻辑，生产回退与 worker 共用）
- `backend/cmd/worker/main.go` — Kafka 消费者

**修改**
- `backend/internal/service/link_service.go` — `LogClick` 改为发 Kafka，失败回退 ClickStore；构造器加 publisher 参数
- `backend/config/config.go` — 加 `KafkaBrokers` / `KafkaTopic`
- `backend/go.mod` — 加 `github.com/segmentio/kafka-go`
- `backend/Dockerfile` — 同时构建 `/server` 与 `/worker`
- `docker-compose.yml` — 加 `kafka` + `kafka-worker` 服务
- `.env.example` — 补 KAFKA_BROKERS / KAFKA_TOPIC 说明

**不在范围内**：前端、GORM、gRPC、数据库 schema、其余 service。

## 组件设计

### 1. `internal/mq/click.go`

```go
type ClickEvent struct {
    LinkID    int64     `json:"link_id"`
    IP        string    `json:"ip"`
    UserAgent string    `json:"user_agent"`
    Platform  string    `json:"platform"`
    Referer   string    `json:"referer"`
    CreatedAt time.Time `json:"created_at"`
}

// ClickPublisher 发布点击事件（生产端接口，便于测试 mock）
type ClickPublisher interface {
    PublishClick(ctx context.Context, e ClickEvent) error
}

// KafkaClickPublisher 基于 segmentio/kafka-go 的实现
type KafkaClickPublisher struct {
    writer *kafka.Writer
}
func NewKafkaClickPublisher(brokers []string, topic string) *KafkaClickPublisher
func (p *KafkaClickPublisher) PublishClick(ctx, e) error  // JSON 序列化后写 topic
func (p *KafkaClickPublisher) Close() error
```

### 2. `internal/service/click_store.go`

```go
// ClickStore 直写点击日志（生产降级回退 + worker 消费共用）
type ClickStore struct { db *pgxpool.Pool }
func NewClickStore(db *pgxpool.Pool) *ClickStore
func (s *ClickStore) WriteClick(ctx, linkID int64, ip, userAgent, platform, referer string) error
// 事务内：INSERT INTO click_logs + UPDATE links SET click_count = click_count + 1
```

### 3. `link_service.go` LogClick 改造

```go
// LogClick 发布点击事件到 Kafka；Kafka 不可用时回退直写
func (s *LinkService) LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) {
    if s.kafka != nil {
        if err := s.kafka.PublishClick(ctx, ClickEvent{...}); err == nil {
            return
        }
        // Kafka 失败 → 落到直写，保证不丢
    }
    _ = s.clickStore.WriteClick(ctx, linkID, ip, userAgent, platform, referer)
}
```

- `LinkService` 增加字段 `kafka ClickPublisher`、`clickStore *ClickStore`
- `NewLinkService(db, baseURL, cache, kafka, clickStore)` — 构造器签名变化，main.go 同步
- `kafka == nil` 即 `KafkaBrokers` 为空 → 永远直写（默认禁用 Kafka 的降级态）

### 4. `cmd/worker/main.go`

- 读环境变量：`DATABASE_URL`、`KAFKA_BROKERS`、`KAFKA_TOPIC`（默认 `clicks`）
- `kafka.Reader`（groupID `click-worker`），循环 `ReadMessage`
- 每条消息反序列化 `ClickEvent` → `ClickStore.WriteClick`（事务）
- 消费失败记日志、不 crash；优雅退出（signal 处理 + `reader.Close`）
- `KAFKA_BROKERS` 为空时直接报错退出（worker 没 Kafka 没意义）

### 5. config 与部署

```go
// config.go 新增
KafkaBrokers string // getEnv("KAFKA_BROKERS", "") 空 = 禁用
KafkaTopic   string // getEnv("KAFKA_TOPIC", "clicks")
```

- docker-compose：
  - `kafka`：`apache/kafka:3.8.0`（或 bitnami/kafka），单节点 KRaft，`PLAINTEXT://:9092`，healthcheck 用 `kafka-topics.sh`/`kafka-broker-api-versions.sh`
  - `kafka-worker`：`build: ./backend`，command 覆盖为 `/worker`，env 加 `KAFKA_BROKERS: kafka:9092`
  - `backend` env 加 `KAFKA_BROKERS: kafka:9092`
- Dockerfile：builder 阶段 `go build -o /server ./cmd/server/` + `go build -o /worker ./cmd/worker/`，runtime 阶段 copy 两个

## 错误处理 / 降级

- `KafkaBrokers` 为空 → `kafka` 字段为 nil → `LogClick` 直写（Kafka 完全关闭也可用）
- `PublishClick` 返回错误 → 回退直写
- worker：单条消费失败记日志继续（不退出）；Kafka 不可达时 reader 重试
- 重定向路径：`go LogClick` 仍在 goroutine 中，请求响应不被 Kafka/DB 阻塞

## 测试

- `internal/mq/click_test.go`：ClickEvent JSON 序列化/反序列化 round-trip
- `internal/service/link_service_test.go` 扩展：`LogClick` 发布成功路径（fake `ClickPublisher`，记录被调用、不触碰 DB）
- 端到端手动验证：
  - `docker compose up kafka backend kafka-worker`（或本地起 kafka）
  - 造一个短链，浏览器访问 `/r/:code` 触发点击
  - 观察 worker 日志消费、`click_logs` 新增记录、`click_count` +1
  - 停掉 kafka 再点击 → 走回退直写，`click_logs` 仍有记录
- `go build ./...`、`go test ./...`、`go vet ./...` 通过

## 简历表述

「将短链点击日志从请求路径解耦为 Kafka 异步消息流：生产端发布 click.event，独立 worker 消费落库；Kafka 故障自动降级直写，保证点击不丢、重定向零阻塞。」
