package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func newTestRunner(modules ...Module) *runner {
	reg := newRegistry()
	for _, m := range modules {
		_ = reg.register(m)
	}
	return &runner{registry: reg, logger: &noopLogger{}}
}

func TestRunner_InitAll_Success(t *testing.T) {
	t.Parallel()
	r := newTestRunner(&mockModule{name: "m1"}, &mockModule{name: "m2"})
	if err := r.initAll(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunner_InitAll_Error(t *testing.T) {
	t.Parallel()
	m := &mockModule{name: "bad", initFn: func(ctx context.Context) error { return errTest }}
	r := newTestRunner(m)
	err := r.initAll(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Errorf("expected module name in error, got %v", err)
	}
}

func TestRunner_StartAll_Success(t *testing.T) {
	t.Parallel()
	r := newTestRunner(&mockModule{name: "m1"}, &mockModule{name: "m2"})
	started, err := r.startAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(started) != 2 {
		t.Errorf("expected 2 started modules, got %d", len(started))
	}
}

func TestRunner_StartAll_ErrorRollback(t *testing.T) {
	t.Parallel()
	stopped := false
	m1 := &mockModule{name: "m1", stopFn: func(ctx context.Context) error {
		stopped = true
		return nil
	}}
	m2 := &mockModule{name: "m2", startFn: func(ctx context.Context) error {
		return errTest
	}}
	r := newTestRunner(m1, m2)
	started, err := r.startAll(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if started != nil {
		t.Errorf("expected nil started, got %v", started)
	}
	if !stopped {
		t.Errorf("expected m1 to be stopped during rollback")
	}
}

func TestRunner_ShutdownModules_Success(t *testing.T) {
	t.Parallel()
	var order []string
	m1 := &mockModule{name: "m1", stopFn: func(ctx context.Context) error {
		order = append(order, "m1")
		return nil
	}}
	m2 := &mockModule{name: "m2", stopFn: func(ctx context.Context) error {
		order = append(order, "m2")
		return nil
	}}
	r := newTestRunner()
	err := r.shutdownModules(context.Background(), []Module{m1, m2})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(order) != 2 || order[0] != "m2" || order[1] != "m1" {
		t.Errorf("expected reverse order [m2 m1], got %v", order)
	}
}

func TestRunner_ShutdownModules_WithErrors(t *testing.T) {
	t.Parallel()
	m := &mockModule{name: "fail", stopFn: func(ctx context.Context) error {
		return errTest
	}}
	r := newTestRunner()
	err := r.shutdownModules(context.Background(), []Module{m})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errTest) {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestRunner_ShutdownModules_Empty(t *testing.T) {
	t.Parallel()
	r := newTestRunner()
	err := r.shutdownModules(context.Background(), nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunner_ShutdownAll(t *testing.T) {
	t.Parallel()
	stopped := false
	m := &mockModule{name: "m1", stopFn: func(ctx context.Context) error {
		stopped = true
		return nil
	}}
	r := newTestRunner(m)
	err := r.shutdownAll(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !stopped {
		t.Errorf("expected module to be stopped")
	}
}
