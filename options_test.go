package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithName_Valid(t *testing.T) {
	t.Parallel()
	a, err := New(WithName("myapp"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.meta.name != "myapp" {
		t.Errorf("expected %q, got %q", "myapp", a.meta.name)
	}
}

func TestWithName_Empty(t *testing.T) {
	t.Parallel()
	_, err := New(WithName(""))
	if !errors.Is(err, ErrAppNameEmpty) {
		t.Errorf("expected ErrAppNameEmpty, got %v", err)
	}
}

func TestWithVersion(t *testing.T) {
	t.Parallel()
	a, err := New(WithVersion("2.0"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.meta.version != "2.0" {
		t.Errorf("expected %q, got %q", "2.0", a.meta.version)
	}
}

func TestWithEnvironment(t *testing.T) {
	t.Parallel()
	a, err := New(WithEnvironment("prod"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.meta.environment != "prod" {
		t.Errorf("expected %q, got %q", "prod", a.meta.environment)
	}
}

func TestWithGracefulTimeout_Valid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		timeout time.Duration
	}{
		{"zero", 0},
		{"positive", 5 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a, err := New(WithGracefulTimeout(tc.timeout))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if a.shutdownTimeout != tc.timeout {
				t.Errorf("expected %v, got %v", tc.timeout, a.shutdownTimeout)
			}
		})
	}
}

func TestWithGracefulTimeout_Negative(t *testing.T) {
	t.Parallel()
	_, err := New(WithGracefulTimeout(-1 * time.Second))
	if !errors.Is(err, ErrShutdownTimeoutNonPositive) {
		t.Errorf("expected ErrShutdownTimeoutNonPositive, got %v", err)
	}
}

func TestWithLogger_Valid(t *testing.T) {
	t.Parallel()
	l := &mockLogger{}
	a, err := New(WithLogger(l))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.logger != l {
		t.Errorf("expected custom logger to be set")
	}
}

func TestWithLogger_Nil(t *testing.T) {
	t.Parallel()
	a, err := New(WithLogger(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := a.logger.(*noopLogger); !ok {
		t.Errorf("expected noopLogger when nil passed, got %T", a.logger)
	}
}

func TestWithHook(t *testing.T) {
	t.Parallel()
	h := Hook{BeforeStart: func(ctx context.Context) error { return nil }}
	a, err := New(WithHook(h))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(a.hooks) != 1 {
		t.Errorf("expected 1 hook, got %d", len(a.hooks))
	}
}
