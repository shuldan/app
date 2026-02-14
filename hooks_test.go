package app

import (
	"context"
	"testing"
)

func TestRunHooksBeforeStart_NilHook(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{}))
	err := a.runHooksBeforeStart(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunHooksBeforeStart_Error(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{
		BeforeStart: func(ctx context.Context) error { return errTest },
	}))
	err := a.runHooksBeforeStart(context.Background())
	if err != errTest {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestRunHooksBeforeStart_Success(t *testing.T) {
	t.Parallel()
	called := false
	a := newTestApp(WithHook(Hook{
		BeforeStart: func(ctx context.Context) error { called = true; return nil },
	}))
	err := a.runHooksBeforeStart(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected hook to be called")
	}
}

func TestRunHooksAfterStart_NilHook(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{}))
	err := a.runHooksAfterStart(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunHooksAfterStart_Error(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{
		AfterStart: func(ctx context.Context) error { return errTest },
	}))
	err := a.runHooksAfterStart(context.Background())
	if err != errTest {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestRunHooksBeforeStop_NilHook(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{}))
	err := a.runHooksBeforeStop(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunHooksBeforeStop_Error(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{
		BeforeStop: func(ctx context.Context) error { return errTest },
	}))
	err := a.runHooksBeforeStop(context.Background())
	if err != errTest {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestRunHooksAfterStop_NilHook(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{}))
	err := a.runHooksAfterStop(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunHooksAfterStop_Error(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{
		AfterStop: func(ctx context.Context) error { return errTest },
	}))
	err := a.runHooksAfterStop(context.Background())
	if err != errTest {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestRunHooksAfterStop_Success(t *testing.T) {
	t.Parallel()
	called := false
	a := newTestApp(WithHook(Hook{
		AfterStop: func(ctx context.Context) error { called = true; return nil },
	}))
	err := a.runHooksAfterStop(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected hook to be called")
	}
}

func TestRunHooks_MultipleHooks(t *testing.T) {
	t.Parallel()
	var order []int
	a := newTestApp(
		WithHook(Hook{BeforeStart: func(ctx context.Context) error { order = append(order, 1); return nil }}),
		WithHook(Hook{BeforeStart: func(ctx context.Context) error { order = append(order, 2); return nil }}),
	)
	err := a.runHooksBeforeStart(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected [1 2], got %v", order)
	}
}

func TestRunHooks_NoHooks(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	if err := a.runHooksBeforeStart(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := a.runHooksAfterStart(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := a.runHooksBeforeStop(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := a.runHooksAfterStop(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
