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
