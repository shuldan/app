package app

import (
	"context"
	"errors"
	"sync"
	"time"
)

type mockModule struct {
	name    string
	initFn  func(ctx context.Context) error
	startFn func(ctx context.Context) error
	stopFn  func(ctx context.Context) error
}

func (m *mockModule) Name() string { return m.name }
func (m *mockModule) Init(ctx context.Context) error {
	if m.initFn != nil {
		return m.initFn(ctx)
	}
	return nil
}
func (m *mockModule) Start(ctx context.Context) error {
	if m.startFn != nil {
		return m.startFn(ctx)
	}
	return nil
}
func (m *mockModule) Stop(ctx context.Context) error {
	if m.stopFn != nil {
		return m.stopFn(ctx)
	}
	return nil
}

type mockBgModule struct {
	mockModule
	errCh chan error
}

func newMockBgModule(name string) *mockBgModule {
	return &mockBgModule{
		mockModule: mockModule{name: name},
		errCh:      make(chan error, 1),
	}
}

func (m *mockBgModule) Err() <-chan error { return m.errCh }

type mockHealthModule struct {
	mockModule
	healthFn func(ctx context.Context) error
}

func (m *mockHealthModule) Health(ctx context.Context) error {
	if m.healthFn != nil {
		return m.healthFn(ctx)
	}
	return nil
}

type mockLogger struct {
	mu    sync.Mutex
	infos []string
	errs  []string
}

func (l *mockLogger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infos = append(l.infos, msg)
}

func (l *mockLogger) Error(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errs = append(l.errs, msg)
}

func quickCancelCtx() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	return ctx, cancel
}

func newTestApp(opts ...Option) *Application {
	a, _ := New(opts...)
	return a
}

var errTest = errors.New("test error")
