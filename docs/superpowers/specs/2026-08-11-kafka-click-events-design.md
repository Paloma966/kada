# Kafka click event stream design

Date: 2026-08-11
Status: confirmed with user

## Goals

Decouple short link click logging from the request path into a Kafka asynchronous message stream:

- The redirect handler no longer writes to the DB directly, but publishes click events to Kafka
- A separate worker consumes the messages, writing `click_logs` + updating `click_count`
- When Kafka is unavailable it automatically falls back to direct writes, so clicks are never lost and the redirect path is unaffected

This is the core highlight of the resume project: **"asynchronous click logging + producer/consumer decoupling + graceful degradation"**.

## Current state (verified)

- `backend/internal/handler/redirect/handler.go`: `go h.svc.LogClick(...)` inside `Redirect` (goroutine direct write)
- `backend/internal/service/link_service.go:486` `LogClick`: `INSERT INTO click_logs` + `UPDATE links SET click_count = click_count + 1`
- The `LinkService` interface (inside the redirect handler) includes `LogClick`; keeping the signature means zero handler changes
- `config/config.go`: `Config` struct + `getEnv`; new fields follow the same pattern
- docker-compose: postgres / redis / backend / frontend / nginx; the backend Dockerfile only builds `/server`
- The data layer still uses pgx (**this design does not introduce GORM / gRPC**)

## Architecture

```
redirect request
   └─ LinkService.LogClick (producer)
        ├─ publish click.event → Kafka topic "clicks"      (normal)
        └─ Kafka unavailable → fall back to direct write click_logs  (degraded, nothing lost)
                    ▼
        Kafka broker (docker single node KRaft)
                    ▼
        cmd/worker (separate consumer process)
        └─ consume → transactionally write click_logs + update click_count
```

## Scope

**New**
- `backend/internal/mq/click.go` — ClickEvent + ClickPublisher interface + Kafka implementation
- `backend/internal/service/click_store.go` — ClickStore (direct write logic, shared by the producer fallback and the worker)
- `backend/cmd/worker/main.go` — Kafka consumer

**Modified**
- `backend/internal/service/link_service.go` — `LogClick` changed to publish to Kafka, falling back to ClickStore on failure; constructor gains a publisher parameter
- `backend/config/config.go` — add `KafkaBrokers` / `KafkaTopic`
- `backend/go.mod` — add `github.com/segmentio/kafka-go`
- `backend/Dockerfile` — build both `/server` and `/worker`
- `docker-compose.yml` — add `kafka` + `kafka-worker` services
- `.env.example` — document KAFKA_BROKERS / KAFKA_TOPIC

**Out of scope**: frontend, GORM, gRPC, database schema, other services.

## Component design

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

// ClickPublisher publishes click events (producer-side interface, easy to mock in tests)
type ClickPublisher interface {
    PublishClick(ctx context.Context, e ClickEvent) error
}

// KafkaClickPublisher is the implementation based on segmentio/kafka-go
type KafkaClickPublisher struct {
    writer *kafka.Writer
}
func NewKafkaClickPublisher(brokers []string, topic string) *KafkaClickPublisher
func (p *KafkaClickPublisher) PublishClick(ctx, e) error  // JSON serialize then write to topic
func (p *KafkaClickPublisher) Close() error
```

### 2. `internal/service/click_store.go`

```go
// ClickStore writes click logs directly (shared by the producer fallback + worker consumption)
type ClickStore struct { db *pgxpool.Pool }
func NewClickStore(db *pgxpool.Pool) *ClickStore
func (s *ClickStore) WriteClick(ctx, linkID int64, ip, userAgent, platform, referer string) error
// inside a transaction: INSERT INTO click_logs + UPDATE links SET click_count = click_count + 1
```

### 3. `link_service.go` LogClick change

```go
// LogClick publishes the click event to Kafka; falls back to a direct write when Kafka is unavailable
func (s *LinkService) LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) {
    if s.kafka != nil {
        if err := s.kafka.PublishClick(ctx, ClickEvent{...}); err == nil {
            return
        }
        // Kafka failed → fall through to direct write, so nothing is lost
    }
    _ = s.clickStore.WriteClick(ctx, linkID, ip, userAgent, platform, referer)
}
```

- `LinkService` gains the fields `kafka ClickPublisher`, `clickStore *ClickStore`
- `NewLinkService(db, baseURL, cache, kafka, clickStore)` — constructor signature changes, main.go updated in sync
- `kafka == nil` means `KafkaBrokers` is empty → always direct write (the degraded state with Kafka fully disabled)

### 4. `cmd/worker/main.go`

- Read env vars: `DATABASE_URL`, `KAFKA_BROKERS`, `KAFKA_TOPIC` (default `clicks`)
- `kafka.Reader` (groupID `click-worker`), loop `ReadMessage`
- Deserialize each message into `ClickEvent` → `ClickStore.WriteClick` (transaction)
- Consumption failures are logged, no crash; graceful shutdown (signal handling + `reader.Close`)
- Exits with an error when `KAFKA_BROKERS` is empty (a worker without Kafka is pointless)

### 5. config and deployment

```go
// config.go additions
KafkaBrokers string // getEnv("KAFKA_BROKERS", "") empty = disabled
KafkaTopic   string // getEnv("KAFKA_TOPIC", "clicks")
```

- docker-compose:
  - `kafka`: `apache/kafka:3.8.0` (or bitnami/kafka), single node KRaft, `PLAINTEXT://:9092`, healthcheck using `kafka-topics.sh`/`kafka-broker-api-versions.sh`
  - `kafka-worker`: `build: ./backend`, command overridden to `/worker`, env adds `KAFKA_BROKERS: kafka:9092`
  - `backend` env adds `KAFKA_BROKERS: kafka:9092`
- Dockerfile: builder stage `go build -o /server ./cmd/server/` + `go build -o /worker ./cmd/worker/`, runtime stage copies both

## Error handling / degradation

- `KafkaBrokers` empty → the `kafka` field is nil → `LogClick` writes directly (works with Kafka fully off)
- `PublishClick` returns an error → fall back to direct write
- worker: a single failed consumption is logged and it continues (no exit); the reader retries while Kafka is unreachable
- redirect path: `go LogClick` is still in a goroutine, so the request response is not blocked by Kafka/DB

## Testing

- `internal/mq/click_test.go`: ClickEvent JSON serialization/deserialization round-trip
- `internal/service/link_service_test.go` extension: `LogClick` success publish path (fake `ClickPublisher`, records the call, does not touch the DB)
- end-to-end manual verification:
  - `docker compose up kafka backend kafka-worker` (or run kafka locally)
  - create a short link, visit `/r/:code` in a browser to trigger a click
  - observe the worker log consuming, a new record in `click_logs`, `click_count` +1
  - stop kafka and click again → takes the direct write fallback, `click_logs` still has the record
- `go build ./...`, `go test ./...`, `go vet ./...` pass

## Resume wording

"Decoupled short link click logging from the request path into a Kafka asynchronous message stream: the producer publishes click.event and a separate worker consumes and persists it; a Kafka failure automatically degrades to a direct write, ensuring clicks are never lost and redirects never block."
