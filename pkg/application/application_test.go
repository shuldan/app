package application

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type MockModule struct {
	registerErr error
	startErr    error
	stopErr     error
	stopFn      func()
	started     bool
	stopped     bool
	mu          sync.Mutex
}

func (m *MockModule) Register(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registerErr
}

func (m *MockModule) Start(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *MockModule) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopErr != nil {
		return m.stopErr
	}
	if m.stopFn != nil {
		m.stopFn()
	}
	m.stopped = true
	return nil
}

func (m *MockModule) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

func (m *MockModule) IsStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

func TestNewApplication(t *testing.T) {
	app := New(
		WithName("test-app"),
		WithVersion("1.0.0"),
		WithEnvironment("test"),
		WithGracefulTimeout(5*time.Second),
	)

	if app.meta.name != "test-app" {
		t.Errorf("expected name 'test-app', got %s", app.meta.name)
	}
	if app.meta.version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", app.meta.version)
	}
	if app.meta.environment != "test" {
		t.Errorf("expected environment 'test', got %s", app.meta.environment)
	}
	if app.shutdownTimeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", app.shutdownTimeout)
	}
}

func TestApplicationRegister(t *testing.T) {
	app := New()
	module := &MockModule{}

	err := app.Register(module)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	modules := app.registry.getAll()
	if len(modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(modules))
	}
	if modules[0] != module {
		t.Error("registered module mismatch")
	}
}

func TestApplicationStartSuccess(t *testing.T) {
	app := New()
	module1 := &MockModule{}
	module2 := &MockModule{}

	err := app.Register(module1)
	if err != nil {
		t.Fatal(err)
	}
	err = app.Register(module2)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = app.start(ctx, cancel)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !module1.IsStarted() || !module2.IsStarted() {
		t.Error("modules should be started")
	}

	if app.meta.startTime.IsZero() {
		t.Error("start time should be set")
	}
}

func TestApplicationStartAlreadyRunning(t *testing.T) {
	app := New()
	module := &MockModule{}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := app.start(ctx, cancel)
	if err != nil {
		t.Fatal(err)
	}

	err = app.start(ctx, cancel)
	if err == nil {
		t.Error("expected error for already running app")
	}
}

func TestApplicationStartModuleRegisterError(t *testing.T) {
	app := New()
	module := &MockModule{registerErr: errors.New("register error")}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := app.start(ctx, cancel)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestApplicationStartModuleStartError(t *testing.T) {
	app := New()
	module1 := &MockModule{}
	module2 := &MockModule{startErr: errors.New("start error")}

	_ = app.Register(module1)
	_ = app.Register(module2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := app.start(ctx, cancel)
	if err == nil {
		t.Error("expected error, got nil")
	}

	if !module1.IsStarted() {
		t.Error("module1 should be started")
	}
	if !module1.IsStopped() {
		t.Error("module1 should be stopped after error")
	}
}

func TestApplicationStopSuccess(t *testing.T) {
	app := New()
	atomic.CompareAndSwapInt32(&app.isRunning, 0, 1)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := app.stop(cancel)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if app.meta.stopTime.IsZero() {
		t.Error("stop time should be set")
	}
}

func TestApplicationStopNotRunning(t *testing.T) {
	app := New()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := app.stop(cancel)
	if err == nil {
		t.Error("expected error for not running app")
	}
}

func TestApplicationRunEmptyRegistry(t *testing.T) {
	app := New()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("expected no error or deadline exceeded, got %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationRunWithModules(t *testing.T) {
	app := New()
	module := &MockModule{}
	_ = app.Register(module)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("expected no error or deadline exceeded, got %v", err)
		}
		if !module.IsStarted() {
			t.Error("module should be started")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationRunWithSignal(t *testing.T) {
	app := New()
	module := &MockModule{}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationGracefulShutdownSuccess(t *testing.T) {
	app := New(WithGracefulTimeout(100 * time.Millisecond))
	module := &MockModule{}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !module.IsStopped() {
			t.Error("module should be stopped")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationGracefulShutdownTimeout(t *testing.T) {
	app := New(WithGracefulTimeout(10 * time.Millisecond))

	slowModule := &MockModule{
		stopErr: errors.New("slow stop"),
	}
	_ = app.Register(slowModule)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected timeout error, got nil")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestRegistryRegister(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	module := &MockModule{}

	err := r.register(module)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	modules := r.getAll()
	if len(modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(modules))
	}
}

func TestRegistryAll(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	module1 := &MockModule{}
	module2 := &MockModule{}

	_ = r.register(module1)
	_ = r.register(module2)

	modules := r.getAll()
	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(modules))
	}
}

func TestRegistryStartAllSuccess(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	module1 := &MockModule{}
	module2 := &MockModule{}

	_ = r.register(module1)
	_ = r.register(module2)

	err := runner.startAll(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !module1.IsStarted() || !module2.IsStarted() {
		t.Error("getAll modules should be started")
	}
}

func TestRegistryStartAllRegisterError(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	module := &MockModule{registerErr: errors.New("register error")}

	_ = r.register(module)

	err := runner.startAll(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRegistryStartAllStartError(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	module1 := &MockModule{}
	module2 := &MockModule{startErr: errors.New("start error")}

	_ = r.register(module1)
	_ = r.register(module2)

	err := runner.startAll(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}

	if !module1.IsStarted() {
		t.Error("module1 should be started")
	}
	if !module1.IsStopped() {
		t.Error("module1 should be stopped after error")
	}
}

func TestRegistryShutdownStarted(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	module1 := &MockModule{}
	module2 := &MockModule{}

	_ = r.register(module1)
	_ = r.register(module2)

	ctx := context.Background()

	_ = module1.Start(ctx)
	_ = module2.Start(ctx)

	runner.shutdownStarted(ctx, 1)

	if !module1.IsStopped() {
		t.Error("module1 should be stopped")
	}
	if module2.IsStopped() {
		t.Error("module2 should not be stopped")
	}
}

func TestRegistryShutdownAllSuccess(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	module1 := &MockModule{}
	module2 := &MockModule{}

	_ = r.register(module1)
	_ = r.register(module2)

	ctx := context.Background()

	_ = module1.Start(ctx)
	_ = module2.Start(ctx)

	err := runner.shutdownAll(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !module1.IsStopped() || !module2.IsStopped() {
		t.Error("getAll modules should be stopped")
	}
}

func TestRegistryShutdownAllWithError(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	err1 := errors.New("stop error 1")
	err2 := errors.New("stop error 2")
	module1 := &MockModule{stopErr: err1}
	module2 := &MockModule{stopErr: err2}

	_ = r.register(module1)
	_ = r.register(module2)

	err := runner.shutdownAll(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}

	var unwrapper interface{ Unwrap() []error }
	if !errors.As(err, &unwrapper) {
		t.Fatalf("error does not support Unwrap []error")
	}

	unwrapped := unwrapper.Unwrap()
	if len(unwrapped) != 2 {
		t.Fatal("expected 2 unwrapped errors")
	}
}

func TestRegistryConcurrency(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	const goroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.register(&MockModule{})
		}()
	}

	wg.Wait()

	modules := r.getAll()
	if len(modules) != goroutines {
		t.Errorf("expected %d modules, got %d", goroutines, len(modules))
	}
}

func TestApplicationRunWithGracefulShutdownWithoutTimeout(t *testing.T) {
	app := New(WithGracefulTimeout(0)) // Disable timeout
	module := &MockModule{}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationRunWithGracefulShutdownError(t *testing.T) {
	app := New(WithGracefulTimeout(100 * time.Millisecond))
	stopError := errors.New("stop error")
	module := &MockModule{stopErr: stopError}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error, got nil")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationSetupSignalHandler(t *testing.T) {
	app := New()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool, 1)
	go func() {
		app.setupSignalHandler(ctx, cancel)
		done <- true
	}()

	time.Sleep(10 * time.Millisecond)
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Skip("cannot find process")
	}
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Skip("cannot send signal")
	}

	select {
	case <-done:

	case <-time.After(100 * time.Millisecond):
		t.Error("signal handler timeout")
	}
}

func TestApplicationSetupSignalHandlerContextDone(t *testing.T) {
	app := New()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool, 1)
	go func() {
		app.setupSignalHandler(ctx, cancel)
		done <- true
	}()

	cancel()

	select {
	case <-done:

	case <-time.After(100 * time.Millisecond):
		t.Error("signal handler timeout")
	}
}

func TestApplicationRunModuleStartError(t *testing.T) {
	app := New()
	startError := errors.New("start error")
	module := &MockModule{startErr: startError}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel() // Ensure we don't hang

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error, got nil")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationStopErrorDuringStartFailure(t *testing.T) {
	app := New()
	startError := errors.New("start error")
	stopError := errors.New("stop error")
	module := &MockModule{
		startErr: startError,
		stopErr:  stopError,
	}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := app.start(ctx, cancel)
	if err == nil {
		t.Error("expected start error, got nil")
	}

	if !errors.Is(err, startError) {
		t.Errorf("expected start error, got %v", err)
	}
}

func TestRegistryShutdownStartedWithZeroCount(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	module := &MockModule{}
	_ = r.register(module)

	runner.shutdownStarted(context.Background(), 0)

	if module.IsStopped() {
		t.Error("module should not be stopped")
	}
}

func TestRegistryShutdownStartedWithNegativeCount(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	module := &MockModule{}
	_ = r.register(module)

	runner.shutdownStarted(context.Background(), -1)

	if module.IsStopped() {
		t.Error("module should not be stopped")
	}
}

func TestRegistryShutdownStartedWithCountGreaterThanModules(t *testing.T) {
	r := &registry{
		modules: make([]Module, 0),
	}
	runner := &runner{registry: r}
	module := &MockModule{}
	_ = r.register(module)

	runner.shutdownStarted(context.Background(), 10)

	if !module.IsStopped() {
		t.Error("module should be stopped")
	}
}

func TestApplicationRunWithContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	app := New()
	module := &MockModule{}
	_ = app.Register(module)

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationRunSignalHandler(t *testing.T) {
	app := New(WithName("test-app"), WithVersion("1.0.0"))
	module := &MockModule{}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if !module.IsStarted() {
			t.Error("module should be started")
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationRunGracefulShutdownTimeout(t *testing.T) {
	app := New(WithGracefulTimeout(10 * time.Millisecond))

	slowModule := &MockModule{
		stopFn: func() {
			time.Sleep(20 * time.Millisecond)
		},
	}
	_ = app.Register(slowModule)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected timeout error, got nil")
		}
		if err != nil && err.Error() != "graceful shutdownAll timed out after 10ms" {
			t.Errorf("expected timeout error message, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("test timeout")
	}
}

func TestApplicationRunWithoutGracefulTimeout(t *testing.T) {
	app := New(WithGracefulTimeout(0)) // Disable graceful shutdown timeout
	module := &MockModule{}
	_ = app.Register(module)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- app.Run(ctx)
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !module.IsStopped() {
			t.Error("module should be stopped")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("test timeout")
	}
}
