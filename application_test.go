package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	t.Parallel()
	a, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.shutdownTimeout != 10*time.Second {
		t.Errorf("expected 10s default timeout, got %v", a.shutdownTimeout)
	}
	if a.runner == nil {
		t.Error("expected runner to be set")
	}
}

func TestNew_OptionError(t *testing.T) {
	t.Parallel()
	_, err := New(WithName(""))
	if err == nil {
		t.Fatal("expected error for empty name option")
	}
}

func TestApplication_Register(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	err := a.Register(&mockModule{name: "mod"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplication_Health_NoModules(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	if err := a.Health(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestApplication_Health_Healthy(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	_ = a.Register(&mockHealthModule{
		mockModule: mockModule{name: "hmod"},
		healthFn:   func(ctx context.Context) error { return nil },
	})
	if err := a.Health(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestApplication_Health_Unhealthy(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	_ = a.Register(&mockHealthModule{
		mockModule: mockModule{name: "sick"},
		healthFn:   func(ctx context.Context) error { return errTest },
	})
	err := a.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplication_Health_MixedModules(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	_ = a.Register(&mockModule{name: "plain"})
	_ = a.Register(&mockHealthModule{
		mockModule: mockModule{name: "hc"},
		healthFn:   func(ctx context.Context) error { return nil },
	})
	if err := a.Health(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplication_Uptime(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	a.meta.startTime = time.Now().Add(-1 * time.Second)
	if d := a.Uptime(); d < 1*time.Second {
		t.Errorf("expected >= 1s, got %v", d)
	}
}

func TestApplication_Run_AlreadyRunning(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	a.isRunning.Store(true)
	err := a.Run(context.Background())
	if !errors.Is(err, ErrApplicationAlreadyRunning) {
		t.Errorf("expected ErrApplicationAlreadyRunning, got %v", err)
	}
}

func TestApplication_Run_InitError(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	_ = a.Register(&mockModule{name: "bad", initFn: func(ctx context.Context) error {
		return errTest
	}})
	err := a.Run(context.Background())
	if !errors.Is(err, errTest) {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestApplication_Run_BeforeStartHookError(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{
		BeforeStart: func(ctx context.Context) error { return errTest },
	}))
	err := a.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplication_Run_StartError(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	_ = a.Register(&mockModule{name: "bad", startFn: func(ctx context.Context) error {
		return errTest
	}})
	err := a.Run(context.Background())
	if !errors.Is(err, errTest) {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestApplication_Run_AfterStartHookError(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{
		AfterStart: func(ctx context.Context) error { return errTest },
	}))
	_ = a.Register(&mockModule{name: "m1"})
	err := a.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplication_Run_ContextCancel(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithGracefulTimeout(5 * time.Second))
	_ = a.Register(&mockModule{name: "m1"})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := a.Run(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestApplication_Run_BackgroundError(t *testing.T) {
	t.Parallel()
	bg := newMockBgModule("bgmod")
	bg.startFn = func(ctx context.Context) error {
		go func() {
			time.Sleep(30 * time.Millisecond)
			bg.errCh <- errTest
		}()
		return nil
	}
	a := newTestApp(WithGracefulTimeout(5 * time.Second))
	_ = a.Register(bg)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := a.Run(ctx)
	_ = err
}

func TestApplication_Run_ShutdownTimeout(t *testing.T) {
	t.Parallel()
	slow := &mockModule{name: "slow", stopFn: func(ctx context.Context) error {
		time.Sleep(500 * time.Millisecond)
		return nil
	}}
	a := newTestApp(WithGracefulTimeout(10 * time.Millisecond))
	_ = a.Register(slow)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	err := a.Run(ctx)
	if !errors.Is(err, ErrGracefulShutdownTimedOut) {
		t.Errorf("expected ErrGracefulShutdownTimedOut, got %v", err)
	}
}

func TestApplication_Run_ZeroTimeout(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithGracefulTimeout(0))
	_ = a.Register(&mockModule{name: "m1"})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := a.Run(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestApplication_Run_IsRunningResets(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = a.Run(ctx)
	if a.isRunning.Load() {
		t.Error("expected isRunning to be false after Run completes")
	}
}

func TestApplication_Run_ConcurrentRunCalls(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = a.Run(ctx)
	}()
	time.Sleep(20 * time.Millisecond)
	go func() {
		defer wg.Done()
		errs[1] = a.Run(ctx)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()
	alreadyRunning := errors.Is(errs[0], ErrApplicationAlreadyRunning) || errors.Is(errs[1], ErrApplicationAlreadyRunning)
	if !alreadyRunning {
		t.Errorf("expected one call to return ErrApplicationAlreadyRunning")
	}
}

func TestApplication_Shutdown_BeforeStopHookError(t *testing.T) {
	t.Parallel()
	l := &mockLogger{}
	a := newTestApp(
		WithLogger(l),
		WithHook(Hook{
			BeforeStop: func(ctx context.Context) error { return errTest },
		}),
	)
	_ = a.Register(&mockModule{name: "m1"})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_ = a.Run(ctx)
	l.mu.Lock()
	defer l.mu.Unlock()
	found := false
	for _, e := range l.errs {
		if e == "before stop hook failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'before stop hook failed' log, got %v", l.errs)
	}
}

func TestApplication_Shutdown_AfterStopHookError(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithHook(Hook{
		AfterStop: func(ctx context.Context) error { return errTest },
	}))
	_ = a.Register(&mockModule{name: "m1"})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := a.Run(ctx)
	if err == nil {
		t.Fatal("expected error from after stop hook")
	}
}

func TestApplication_Shutdown_ModuleStopError(t *testing.T) {
	t.Parallel()
	a := newTestApp(WithGracefulTimeout(5 * time.Second))
	_ = a.Register(&mockModule{name: "bad", stopFn: func(ctx context.Context) error {
		return errTest
	}})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := a.Run(ctx)
	if !errors.Is(err, errTest) {
		t.Errorf("expected errTest, got %v", err)
	}
}
