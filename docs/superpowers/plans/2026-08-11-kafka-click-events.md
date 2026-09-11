# Kafka Click Event Stream Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decouple short-link click logging into an asynchronous Kafka message stream: `LinkService.LogClick` publishes `click.event` to Kafka, and a standalone `cmd/worker` consumes and persists it; when Kafka is unavailable it automatically falls back to a direct write.

**Architecture:** Add `internal/mq` (ClickEvent + ClickPublisher interface + a segmentio/kafka-go implementation), `internal/service/click_store.go` (the direct-write logic shared by the production fallback and the worker), and `cmd/worker` (the consumer). `LinkService.LogClick` becomes "publish to Kafka → fall back to a direct write on failure"; the handler signature is unchanged (`go LogClick` keeps a goroutine so the request is not blocked). docker-compose gains a Kafka broker + worker service.

**Tech Stack:** Go 1.26, gin, pgx/v5, segmentio/kafka-go (pure Go, no CGO).

**Spec:** `docs/superpowers/specs/2026-08-11-kafka-click-events-design.md`

## Global Constraints

- Backend verification commands (under `backend/`): `go build ./...`, `go test ./...`, `go vet ./...` must pass. The frontend is untouched.
- Keep the data layer on pgx; **do not introduce GORM / gRPC** (the user has already confirmed these are cut).
- `LogClick`'s public signature and behavioral contract are unchanged: void, fire-and-forget, never blocking or breaking the redirect path.
- Empty `KafkaBrokers` → Kafka fully disabled → direct write is used (the default safe state).
- Resume talking points (the implementation must preserve these): producer/consumer decoupling, automatic degradation to a direct write when Kafka fails, worker as a separate process.
- Follow the existing code style (Chinese comments, service layering, interface injection for testability).
- CI: `.github/workflows/ci.yml` has `backend-lint` (golangci-lint) and `backend-test`. Changed files must pass `golangci-lint` (verify locally with `golangci-lint run`).

---

### Task 1: `internal/mq` — ClickEvent + Kafka publisher (TDD)

**Files:**
- Create: `backend/internal/mq/click.go`
- Test: `backend/internal/mq/click_test.go`

**Interfaces:**
- Consumes: `github.com/segmentio/kafka-go`
- Produces:
  - `type ClickEvent struct { LinkID int64; IP string; UserAgent string; Platform string; Referer string; CreatedAt time.Time }` (lowercase json tags)
  - `type ClickPublisher interface { PublishClick(ctx context.Context, e ClickEvent) error }`
  - `NewKafkaClickPublisher(brokers []string, topic string) *KafkaClickPublisher` — returns nil when brokers is empty
  - `(*KafkaClickPublisher).PublishClick(ctx, e) error` — writes to the topic after JSON serialization
  - `(*KafkaClickPublisher).Close() error`

- [ ] **Step 1: Add the dependency**

```bash
cd backend
go get github.com/segmentio/kafka-go@latest
go mod tidy
go build ./...
```

Expected: compiles successfully, kafka-go appears in go.mod.

- [ ] **Step 2: Write a failing test**

Create `backend/internal/mq/click_test.go`:

```go
package mq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type fakeWriter struct {
	got []kafka.Message
}

func (f *fakeWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.got = append(f.got, msgs...)
	return nil
}

func TestClickEventRoundTrip(t *testing.T) {
	e := ClickEvent{
		LinkID: 42, IP: "1.2.3.4", UserAgent: "Mozilla/5.0",
		Platform: "wechat", Referer: "https://x.com",
		CreatedAt: time.Unix(1700000000, 0).UTC(),
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got ClickEvent
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != e {
		t.Fatalf("round trip mismatch: got %+v want %+v", got, e)
	}
}

func TestNewKafkaClickPublisher_NoBrokersReturnsNil(t *testing.T) {
	if got := NewKafkaClickPublisher(nil, "clicks"); got != nil {
		t.Fatalf("expected nil for empty brokers, got %+v", got)
	}
	if got := NewKafkaClickPublisher([]string{"  ", ""}, "clicks"); got != nil {
		t.Fatalf("expected nil for blank brokers, got %+v", got)
	}
}

func TestKafkaClickPublisher_PublishClickWritesJSON(t *testing.T) {
	w := &fakeWriter{}
	p := &KafkaClickPublisher{writer: w}
	e := ClickEvent{LinkID: 7, IP: "9.9.9.9", Platform: "browser", CreatedAt: time.Unix(1700000000, 0).UTC()}
	if err := p.PublishClick(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(w.got) != 1 {
		t.Fatalf("expected 1 message, got %d", len(w.got))
	}
	var decoded ClickEvent
	if err := json.Unmarshal(w.got[0].Value, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != e {
		t.Fatalf("decoded mismatch: got %+v want %+v", decoded, e)
	}
}
```

- [ ] **Step 3: Run the test and confirm it fails**

Run: `cd backend && go test ./internal/mq/ -run TestClickEvent -count=1`

Expected: FAIL — `cannot find package` or `undefined: ClickEvent`.

- [ ] **Step 4: Implement**

Create `backend/internal/mq/click.go`:

```go
package mq

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// ClickEvent is a short-link click event (the message payload shared by producer and consumer)
type ClickEvent struct {
	LinkID    int64     `json:"link_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Platform  string    `json:"platform"`
	Referer   string    `json:"referer"`
	CreatedAt time.Time `json:"created_at"`
}

// ClickPublisher publishes click events (producer-side interface, for test mocks and fallback)
type ClickPublisher interface {
	PublishClick(ctx context.Context, e ClickEvent) error
}

// messageWriter narrows kafka.Writer's write interface so unit tests can inject a fake
type messageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

// KafkaClickPublisher is the segmentio/kafka-go based implementation
type KafkaClickPublisher struct {
	writer messageWriter
}

// NewKafkaClickPublisher constructs a publisher; returns nil when brokers is empty (meaning Kafka is disabled)
func NewKafkaClickPublisher(brokers []string, topic string) *KafkaClickPublisher {
	var valid []string
	for _, b := range brokers {
		if b = strings.TrimSpace(b); b != "" {
			valid = append(valid, b)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	return &KafkaClickPublisher{
		writer: kafka.NewWriter(kafka.WriterConfig{
			Brokers:      valid,
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
		}),
	}
}

// PublishClick serializes the ClickEvent and writes it to the topic
func (p *KafkaClickPublisher) PublishClick(ctx context.Context, e ClickEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Value: b})
}

// Close closes the underlying writer
func (p *KafkaClickPublisher) Close() error {
	if w, ok := p.writer.(*kafka.Writer); ok {
		return w.Close()
	}
	return nil
}
```

- [ ] **Step 5: Run the test and confirm it passes**

Run: `cd backend && go test ./internal/mq/ -count=1`

Expected: PASS, all 3 tests pass.

- [ ] **Step 6: lint + vet**

Run: `cd backend && go vet ./internal/mq/ && golangci-lint run ./internal/mq/... 2>/dev/null || echo "(golangci-lint not available locally, vet passed)"`

Expected: vet passes; if golangci-lint is not installed locally, skip it (CI will run it).

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/mq/
git commit -m "feat: click event model and kafka publisher (segmentio/kafka-go)"
```

---

### Task 2: ClickStore + LinkService wired into Kafka + config (TDD)

**Files:**
- Create: `backend/internal/service/click_store.go`
- Test: `backend/internal/service/click_store_test.go`
- Modify: `backend/internal/service/link_service.go`
- Modify: `backend/config/config.go`
- Modify: `backend/cmd/server/main.go`

**Interfaces:**
- Consumes: `mq.ClickPublisher` (Task 1), `mq.ClickEvent`
- Produces:
  - `type ClickWriter interface { WriteClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) error }`
  - `type ClickStore struct{ db *pgxpool.Pool }`; `NewClickStore(db *pgxpool.Pool) *ClickStore`; `(*ClickStore).WriteClick(...)` satisfies `ClickWriter`
  - `service.NewLinkService(db *pgxpool.Pool, baseURL string, cache *CacheService, kafka mq.ClickPublisher, clickWriter ClickWriter) *LinkService` (**signature change**)
  - `config.Config` gains the fields `KafkaBrokers` and `KafkaTopic`, plus the method `Brokers() []string`

- [ ] **Step 1: Write a failing test**

Create `backend/internal/service/click_store_test.go`:

```go
package service

import (
	"context"
	"testing"

	"github.com/chun/kada-backend/internal/mq"
)

type fakePublisher struct {
	calls int
	err   error
}

func (f *fakePublisher) PublishClick(_ context.Context, _ mq.ClickEvent) error {
	f.calls++
	return f.err
}

type fakeWriter struct {
	calls int
}

func (f *fakeWriter) WriteClick(_ context.Context, _ int64, _, _, _, _ string) error {
	f.calls++
	return nil
}

func TestLogClick_PublishSuccess(t *testing.T) {
	pub := &fakePublisher{}
	svc := &LinkService{kafka: pub, clickWriter: &fakeWriter{}}
	svc.LogClick(context.Background(), 1, "1.2.3.4", "ua", "browser", "ref")
	if pub.calls != 1 {
		t.Fatalf("expected publisher called once, got %d", pub.calls)
	}
}

func TestLogClick_PublishErrorFallsBack(t *testing.T) {
	pub := &fakePublisher{err: assertErr("kafka down")}
	w := &fakeWriter{}
	svc := &LinkService{kafka: pub, clickWriter: w}
	svc.LogClick(context.Background(), 1, "1.2.3.4", "ua", "browser", "ref")
	if pub.calls != 1 || w.calls != 1 {
		t.Fatalf("expected publish 1 + fallback 1, got publish=%d write=%d", pub.calls, w.calls)
	}
}

func TestLogClick_NoKafkaDirectWrite(t *testing.T) {
	w := &fakeWriter{}
	svc := &LinkService{kafka: nil, clickWriter: w}
	svc.LogClick(context.Background(), 1, "1.2.3.4", "ua", "browser", "ref")
	if w.calls != 1 {
		t.Fatalf("expected direct write, got %d", w.calls)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
```

Note: these tests construct a `LinkService` literal directly (the fields `kafka` and `clickWriter` are visible), so **no DB is needed**.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd backend && go test ./internal/service/ -run TestLogClick -count=1`

Expected: FAIL — `unknown field 'kafka' in struct literal`.

- [ ] **Step 3: Implement ClickStore**

Create `backend/internal/service/click_store.go`:

```go
package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ClickWriter writes click logs directly (shared by the production fallback and worker consumption)
type ClickWriter interface {
	WriteClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) error
}

// ClickStore is the pgx implementation of ClickWriter
type ClickStore struct {
	db *pgxpool.Pool
}

func NewClickStore(db *pgxpool.Pool) *ClickStore {
	return &ClickStore{db: db}
}

// WriteClick inside a transaction: insert the click log + increment the counter
func (s *ClickStore) WriteClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO click_logs (link_id, ip, user_agent, platform, referer)
		VALUES ($1, $2, $3, $4, $5)
	`, linkID, ip, userAgent, platform, referer); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE links SET click_count = click_count + 1 WHERE id = $1
	`, linkID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
```

- [ ] **Step 4: Rework link_service.go**

In `backend/internal/service/link_service.go`:

(1) Add to the imports at the top of the file:

```go
"github.com/chun/kada-backend/internal/mq"
```

(2) Add two fields to the `LinkService` struct:

```go
type LinkService struct {
	db         *pgxpool.Pool
	baseURL    string
	cache      *CacheService
	kafka      mq.ClickPublisher // Kafka publisher; nil means disabled
	clickWriter ClickWriter      // direct write (used for fallback)
}
```

(3) Constructor signature and assignments:

```go
func NewLinkService(db *pgxpool.Pool, baseURL string, cache *CacheService, kafka mq.ClickPublisher, clickWriter ClickWriter) *LinkService {
	return &LinkService{db: db, baseURL: baseURL, cache: cache, kafka: kafka, clickWriter: clickWriter}
}
```

(4) Rewrite `LogClick` as "publish to Kafka → fall back to a direct write on failure":

```go
// LogClick publishes the click event to Kafka; falls back to a direct write when Kafka is unavailable, so clicks are not lost
func (s *LinkService) LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) {
	if s.kafka != nil {
		err := s.kafka.PublishClick(ctx, mq.ClickEvent{
			LinkID:    linkID,
			IP:        ip,
			UserAgent: userAgent,
			Platform:  platform,
			Referer:   referer,
			CreatedAt: time.Now(),
		})
		if err == nil {
			return
		}
		// Kafka failed → fall through to the direct write
	}
	if s.clickWriter != nil {
		_ = s.clickWriter.WriteClick(ctx, linkID, ip, userAgent, platform, referer)
	}
}
```

(Confirm that link_service.go already imports `time`; add it if not.)

- [ ] **Step 5: Add Kafka config**

`backend/config/config.go`:

(1) Add fields to the struct:

```go
	// Kafka (click event stream; empty = disabled)
	KafkaBrokers string
	KafkaTopic   string
```

(2) Add to `Load()`:

```go
		KafkaBrokers:      getEnv("KAFKA_BROKERS", ""),
		KafkaTopic:        getEnv("KAFKA_TOPIC", "clicks"),
```

(3) Add the method (import `strings` at the top of the file):

```go
// Brokers splits the comma-separated broker list, trimming whitespace and empty entries
func (c *Config) Brokers() []string {
	var out []string
	for _, b := range strings.Split(c.KafkaBrokers, ",") {
		if b = strings.TrimSpace(b); b != "" {
			out = append(out, b)
		}
	}
	return out
}
```

- [ ] **Step 6: Wire up main.go**

`backend/cmd/server/main.go`:

```go
	// Kafka click event publisher (returns nil when there are no brokers = disabled)
	kafkaPub := mq.NewKafkaClickPublisher(cfg.Brokers(), cfg.KafkaTopic)
	linkSvc := service.NewLinkService(db, cfg.BaseURL, cacheSvc, kafkaPub, service.NewClickStore(db))
```

Also add `"github.com/chun/kada-backend/internal/mq"` to the imports (introduce it if Task 1's mq usage did not already — main.go does not reference it yet, so it is a new import).

- [ ] **Step 7: Full test + build + vet**

Run: `cd backend && go test ./... -count=1 && go build ./... && go vet ./...`

Expected: everything PASSES (including the existing tests); build/vet pass.

- [ ] **Step 8: Commit**

```bash
git add internal/service/click_store.go internal/service/click_store_test.go internal/service/link_service.go config/config.go cmd/server/main.go
git commit -m "feat: route click logging through kafka publisher with direct-write fallback"
```

---

### Task 3: `cmd/worker` — Kafka consumer (TDD)

**Files:**
- Create: `backend/cmd/worker/main.go`
- Test: `backend/cmd/worker/main_test.go`

**Interfaces:**
- Consumes: `service.ClickWriter`, `service.NewClickStore`, `infra.NewDB`, `mq.ClickEvent`
- Produces: `processClickMessage(msg []byte, writer service.ClickWriter) error` (testable); `main()` orchestrates the reader loop

- [ ] **Step 1: Write a failing test**

Create `backend/cmd/worker/main_test.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chun/kada-backend/internal/mq"
)

type recorderWriter struct {
	got []mq.ClickEvent
}

func (r *recorderWriter) WriteClick(_ context.Context, linkID int64, ip, ua, platform, referer string) error {
	r.got = append(r.got, mq.ClickEvent{LinkID: linkID, IP: ip, UserAgent: ua, Platform: platform, Referer: referer})
	return nil
}

func TestProcessClickMessage_Valid(t *testing.T) {
	w := &recorderWriter{}
	e := mq.ClickEvent{LinkID: 5, IP: "8.8.8.8", UserAgent: "ua", Platform: "qq", Referer: "r", CreatedAt: time.Unix(1700000000, 0).UTC()}
	b, _ := json.Marshal(e)
	if err := processClickMessage(b, w); err != nil {
		t.Fatal(err)
	}
	if len(w.got) != 1 || w.got[0].LinkID != 5 {
		t.Fatalf("expected 1 write with link 5, got %+v", w.got)
	}
}

func TestProcessClickMessage_InvalidJSON(t *testing.T) {
	w := &recorderWriter{}
	if err := processClickMessage([]byte("{not json"), w); err == nil {
		t.Fatal("expected error for invalid json")
	}
	if len(w.got) != 0 {
		t.Fatalf("expected no write on invalid json, got %d", len(w.got))
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `cd backend && go test ./cmd/worker/ -count=1`

Expected: FAIL — `undefined: processClickMessage`.

- [ ] **Step 3: Implement**

Create `backend/cmd/worker/main.go`:

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"

	"github.com/chun/kada-backend/internal/infra"
	"github.com/chun/kada-backend/internal/mq"
	"github.com/chun/kada-backend/internal/service"
)

// processClickMessage deserializes one click message and persists it
func processClickMessage(msg []byte, writer service.ClickWriter) error {
	var e mq.ClickEvent
	if err := json.Unmarshal(msg, &e); err != nil {
		return err
	}
	return writer.WriteClick(context.Background(), e.LinkID, e.IP, e.UserAgent, e.Platform, e.Referer)
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	brokers := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "clicks"
	}
	if brokers == "" {
		log.Fatal("KAFKA_BROKERS is required for worker")
	}

	db, err := infra.NewDB(databaseURL)
	if err != nil {
		log.Fatalf("database connect failed: %v", err)
	}
	defer infra.CloseDB(db)

	store := service.NewClickStore(db)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokers},
		Topic:    topic,
		GroupID:  "click-worker",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("🧵 click-worker consuming topic %q from %s", topic, brokers)
	for {
		m, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("shutdown signal received")
				return
			}
			log.Printf("read message failed: %v", err)
			continue
		}
		if err := processClickMessage(m.Value, store); err != nil {
			log.Printf("process message failed: %v", err)
			continue
		}
	}
}
```

- [ ] **Step 4: Run the test and confirm it passes**

Run: `cd backend && go test ./cmd/worker/ -count=1`

Expected: PASS.

- [ ] **Step 5: build + vet + lint**

Run: `cd backend && go build ./... && go vet ./...`

Expected: passes.

- [ ] **Step 6: Commit**

```bash
git add cmd/worker/
git commit -m "feat: kafka click consumer worker"
```

---

### Task 4: Docker + compose + env wiring

**Files:**
- Modify: `backend/Dockerfile`
- Modify: `docker-compose.yml` (repository root)
- Modify: `.env.example`

**Interfaces:**
- Consumes: `cmd/worker` (the `/worker` binary from Task 3)
- Produces: `docker compose config` parses; the `kafka` and `kafka-worker` services are ready

- [ ] **Step 1: Build two binaries in the Dockerfile**

`backend/Dockerfile` builder stage:

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server/ \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker ./cmd/worker/
```

runtime stage:

```dockerfile
COPY --from=builder /server /server
COPY --from=builder /worker /worker
```

- [ ] **Step 2: Add kafka and kafka-worker to docker-compose**

Add the following under `services:` in `docker-compose.yml`:

```yaml
  kafka:
    image: apache/kafka:3.8.0
    ports:
      - "127.0.0.1:9092:9092"
    environment:
      KAFKA_NODE_ID: "1"
      KAFKA_PROCESS_ROLES: "broker,controller"
      KAFKA_LISTENERS: "PLAINTEXT://:9092,CONTROLLER://:9093"
      KAFKA_ADVERTISED_LISTENERS: "PLAINTEXT://kafka:9092"
      KAFKA_CONTROLLER_LISTENER_NAMES: "CONTROLLER"
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT"
      KAFKA_CONTROLLER_QUORUM_VOTERS: "1@kafka:9093"
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: "1"
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: "1"
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: "0"
    healthcheck:
      test: ["CMD", "/opt/kafka/bin/kafka-broker-api-versions.sh", "--bootstrap-server", "localhost:9092"]
      interval: 10s
      timeout: 5s
      retries: 10

  kafka-worker:
    build: ./backend
    command: ["/worker"]
    environment:
      DATABASE_URL: postgres://kada:kada123@postgres:5432/kada?sslmode=disable
      KAFKA_BROKERS: kafka:9092
      KAFKA_TOPIC: clicks
    depends_on:
      postgres:
        condition: service_healthy
      kafka:
        condition: service_healthy
    restart: unless-stopped
```

And add the following to the `backend` service's `environment:`:

```yaml
      KAFKA_BROKERS: kafka:9092
      KAFKA_TOPIC: clicks
```

- [ ] **Step 3: Add notes to .env.example**

Append:

```bash
# ========== Kafka (click event stream) ==========
# Comma-separated brokers; leave empty = disable Kafka and write clicks directly to the database
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=clicks
```

- [ ] **Step 4: Verification**

```bash
cd /home/chun/dev/projects/kada && docker compose config --quiet && echo "compose OK"
cd backend && go build ./...
```

Expected: compose parses successfully; backend builds.

(Optional end-to-end, requires docker: after `docker compose up -d kafka`, start backend + worker, visit `/r/:code` to trigger a click, and the worker log should show consumption.)

- [ ] **Step 5: Commit**

```bash
git add backend/Dockerfile docker-compose.yml .env.example
git commit -m "feat: kafka broker + click worker in docker compose"
```

---

## Self-check record

- **Spec coverage**: ClickEvent/publisher (Task 1), ClickStore + LogClick production fallback (Task 2), config disabled state (Task 2), worker consumption (Task 3), docker/worker deployment (Task 4), degradation to a direct write (covered by Task 2 tests), tests and resume talking points (each task).
- **Placeholder scan**: no TBD/TODO; the code is complete.
- **Type consistency**: `mq.ClickPublisher` / `mq.ClickEvent` / `NewKafkaClickPublisher` / `ClickWriter` / `NewClickStore` / `WriteClick` / the new `NewLinkService` signature / `Config.Brokers` are defined in Tasks 1-2 and consumed in Tasks 2-4, with matching signatures; `cmd/worker` consumes `service.ClickWriter` and `mq.ClickEvent`.
