package app

import (
	"context"
	"testing"
	"time"
)

func TestMeta_EnrichContext(t *testing.T) {
	t.Parallel()
	m := meta{
		name:        "testapp",
		version:     "1.0.0",
		environment: "testing",
		startTime:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	ctx := m.enrichContext(context.Background())
	if got := NameFromContext(ctx); got != "testapp" {
		t.Errorf("expected %q, got %q", "testapp", got)
	}
	if got := VersionFromContext(ctx); got != "1.0.0" {
		t.Errorf("expected %q, got %q", "1.0.0", got)
	}
	if got := EnvironmentFromContext(ctx); got != "testing" {
		t.Errorf("expected %q, got %q", "testing", got)
	}
	if got := StartTimeFromContext(ctx); !got.Equal(m.startTime) {
		t.Errorf("expected %v, got %v", m.startTime, got)
	}
}

func TestMeta_Uptime_StopTimeZero(t *testing.T) {
	t.Parallel()
	m := meta{startTime: time.Now().Add(-100 * time.Millisecond)}
	d := m.uptime()
	if d < 100*time.Millisecond {
		t.Errorf("expected uptime >= 100ms, got %v", d)
	}
}

func TestMeta_Uptime_StopTimeSet(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	stop := start.Add(5 * time.Second)
	m := meta{startTime: start, stopTime: stop}
	if d := m.uptime(); d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
}

func TestNameFromContext_Empty(t *testing.T) {
	t.Parallel()
	if got := NameFromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestVersionFromContext_Empty(t *testing.T) {
	t.Parallel()
	if got := VersionFromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestEnvironmentFromContext_Empty(t *testing.T) {
	t.Parallel()
	if got := EnvironmentFromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestStartTimeFromContext_Zero(t *testing.T) {
	t.Parallel()
	if got := StartTimeFromContext(context.Background()); !got.IsZero() {
		t.Errorf("expected zero time, got %v", got)
	}
}
