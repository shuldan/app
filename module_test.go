package app

import (
	"context"
	"testing"
)

func TestMockModule_ImplementsModule(t *testing.T) {
	t.Parallel()
	var _ Module = &mockModule{}
}

func TestMockBgModule_ImplementsBackgroundModule(t *testing.T) {
	t.Parallel()
	var _ BackgroundModule = &mockBgModule{}
}

func TestMockHealthModule_ImplementsHealthChecker(t *testing.T) {
	t.Parallel()
	var _ HealthChecker = &mockHealthModule{}
}

func TestBackgroundModule_ErrChannel(t *testing.T) {
	t.Parallel()
	bg := newMockBgModule("bg")
	bg.errCh <- errTest
	err := <-bg.Err()
	if err != errTest {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestBackgroundModule_ErrChannelClose(t *testing.T) {
	t.Parallel()
	bg := newMockBgModule("bg")
	close(bg.errCh)
	_, ok := <-bg.Err()
	if ok {
		t.Error("expected channel to be closed")
	}
}

func TestCollectBackgroundErrors_NoBgModules(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	_ = a.Register(&mockModule{name: "plain"})
	ch := a.collectBackgroundErrors()
	if ch != nil {
		t.Error("expected nil channel when no bg modules")
	}
}

func TestCollectBackgroundErrors_WithError(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	bg := newMockBgModule("bg1")
	_ = a.Register(bg)
	ch := a.collectBackgroundErrors()
	bg.errCh <- errTest
	err := <-ch
	if err == nil {
		t.Fatal("expected error from merged channel")
	}
}

func TestCollectBackgroundErrors_ChannelCloseNilErr(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	bg := newMockBgModule("bg1")
	_ = a.Register(bg)
	ch := a.collectBackgroundErrors()
	close(bg.errCh)
	for range ch {
	}
}

func TestCollectBackgroundErrors_MultipleModules(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	bg1 := newMockBgModule("bg1")
	bg2 := newMockBgModule("bg2")
	_ = a.Register(bg1)
	_ = a.Register(bg2)
	ch := a.collectBackgroundErrors()
	bg1.errCh <- errTest
	close(bg2.errCh)
	count := 0
	for range ch {
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 error, got %d", count)
	}
}

func TestHealthChecker_MultipleErrors(t *testing.T) {
	t.Parallel()
	a := newTestApp()
	_ = a.Register(&mockHealthModule{
		mockModule: mockModule{name: "h1"},
		healthFn:   func(ctx context.Context) error { return errTest },
	})
	_ = a.Register(&mockHealthModule{
		mockModule: mockModule{name: "h2"},
		healthFn:   func(ctx context.Context) error { return errTest },
	})
	err := a.Health(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
