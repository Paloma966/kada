package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/chun/kada-backend/internal/mq"
)

type assertErr string

func (e assertErr) Error() string { return string(e) }

type recorderWriter struct {
	got []mq.ClickEvent
}

func (r *recorderWriter) WriteClick(_ context.Context, linkID int64, ip, ua, platform, referer string, createdAt time.Time) error {
	r.got = append(r.got, mq.ClickEvent{LinkID: linkID, IP: ip, UserAgent: ua, Platform: platform, Referer: referer, CreatedAt: createdAt})
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
	if !w.got[0].CreatedAt.Equal(e.CreatedAt) {
		t.Fatalf("expected CreatedAt to be preserved, got %v want %v", w.got[0].CreatedAt, e.CreatedAt)
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

func TestIsPermanentError(t *testing.T) {
	// 非法 JSON → 永久性错误
	if err := processClickMessage([]byte("{bad"), &recorderWriter{}); !isPermanentError(err) {
		t.Error("processClickMessage invalid json should be classified permanent")
	}

	// 外键违反（23503，如链接已删除）→ 永久性错误
	fkErr := &pgconn.PgError{Code: "23503", Message: "violates foreign key constraint"}
	if !isPermanentError(fkErr) {
		t.Error("foreign key violation should be permanent")
	}

	// 连接类错误（08006 连接丢失）→ 可重试
	connErr := &pgconn.PgError{Code: "08006", Message: "connection lost"}
	if isPermanentError(connErr) {
		t.Error("connection failure should be retryable")
	}

	// 普通错误 → 可重试
	if isPermanentError(assertErr("boom")) {
		t.Error("plain error should be retryable")
	}
}

func TestAttemptTracker(t *testing.T) {
	tr := newAttemptTracker()
	if n := tr.record("clicks", 0, 42); n != 1 {
		t.Errorf("expected attempt 1, got %d", n)
	}
	if n := tr.record("clicks", 0, 42); n != 2 {
		t.Errorf("expected attempt 2, got %d", n)
	}
	tr.reset("clicks", 0, 42)
	if n := tr.record("clicks", 0, 42); n != 1 {
		t.Errorf("expected attempt 1 after reset, got %d", n)
	}
}
