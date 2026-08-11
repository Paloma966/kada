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
	w := &fakeWriter{}
	svc := &LinkService{kafka: pub, clickWriter: w}
	svc.LogClick(context.Background(), 1, "1.2.3.4", "ua", "browser", "ref")
	if pub.calls != 1 {
		t.Fatalf("expected publisher called once, got %d", pub.calls)
	}
	if w.calls != 0 {
		t.Fatalf("expected no direct write when publish succeeds, got %d", w.calls)
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
