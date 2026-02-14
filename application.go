package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Application struct {
	meta            meta
	registry        *registry
	runner          *runner
	logger          Logger
	hooks           []Hook
	isRunning       atomic.Bool
	shutdownTimeout time.Duration
}

func New(opts ...Option) (*Application, error) {
	reg := newRegistry()

	a := &Application{
		registry:        reg,
		logger:          &noopLogger{},
		shutdownTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		if err := opt(a); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	a.runner = &runner{
		registry: reg,
		logger:   a.logger,
	}

	return a, nil
}

func (a *Application) Register(module Module) error {
	return a.registry.register(module)
}

func (a *Application) Health(ctx context.Context) error {
	var errs []error
	for _, m := range a.registry.getAll() {
		if hc, ok := m.(HealthChecker); ok {
			if err := hc.Health(ctx); err != nil {
				errs = append(errs, fmt.Errorf("module %q: %w", m.Name(), err))
			}
		}
	}
	return errors.Join(errs...)
}

func (a *Application) Uptime() time.Duration {
	return a.meta.uptime()
}

func (a *Application) Run(ctx context.Context) error {
	if !a.isRunning.CompareAndSwap(false, true) {
		return ErrApplicationAlreadyRunning
	}

	a.registry.lock()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.meta.startTime = time.Now()
	ctx = a.meta.enrichContext(ctx)

	go a.setupSignalHandler(ctx, cancel)

	a.logger.Info("initializing modules")
	if err := a.runner.initAll(ctx); err != nil {
		return err
	}

	if err := a.runHooksBeforeStart(ctx); err != nil {
		return fmt.Errorf("before start hook: %w", err)
	}

	a.logger.Info("starting modules")
	startedModules, err := a.runner.startAll(ctx)
	if err != nil {
		return err
	}

	if err := a.runHooksAfterStart(ctx); err != nil {
		a.logger.Error("after start hook failed, shutting down", "error", err)
		shutdownErr := a.runner.shutdownModules(context.Background(), startedModules)
		return errors.Join(fmt.Errorf("after start hook: %w", err), shutdownErr)
	}

	bgErrCh := a.collectBackgroundErrors()

	a.logger.Info("application started")

	select {
	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
	case bgErr := <-bgErrCh:
		a.logger.Error("background module failed", "error", bgErr)
		cancel()
	}

	return a.shutdown()
}

func (a *Application) shutdown() error {
	defer func() {
		a.meta.stopTime = time.Now()
		a.isRunning.Store(false)
	}()

	hookCtx := context.Background()
	if err := a.runHooksBeforeStop(hookCtx); err != nil {
		a.logger.Error("before stop hook failed", "error", err)
	}

	var shutdownErr error
	if a.shutdownTimeout > 0 {
		shutdownCtx, timeoutCancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer timeoutCancel()

		errCh := make(chan error, 1)
		go func() {
			errCh <- a.runner.shutdownAll(shutdownCtx)
		}()

		select {
		case shutdownErr = <-errCh:
		case <-shutdownCtx.Done():
			shutdownErr = ErrGracefulShutdownTimedOut
		}
	} else {
		shutdownErr = a.runner.shutdownAll(context.Background())
	}

	if shutdownErr != nil {
		a.logger.Error("shutdown completed with errors", "error", shutdownErr)
	} else {
		a.logger.Info("shutdown completed successfully")
	}

	if err := a.runHooksAfterStop(hookCtx); err != nil {
		a.logger.Error("after stop hook failed", "error", err)
		shutdownErr = errors.Join(shutdownErr, fmt.Errorf("after stop hook: %w", err))
	}

	return shutdownErr
}

func (a *Application) collectBackgroundErrors() <-chan error {
	modules := a.registry.getAll()

	var bgModules []BackgroundModule
	for _, m := range modules {
		if bg, ok := m.(BackgroundModule); ok {
			bgModules = append(bgModules, bg)
		}
	}

	if len(bgModules) == 0 {
		return nil
	}

	merged := make(chan error, len(bgModules))
	var wg sync.WaitGroup

	for _, bg := range bgModules {
		wg.Add(1)
		go func(bg BackgroundModule) {
			defer wg.Done()
			if err, ok := <-bg.Err(); ok && err != nil {
				merged <- fmt.Errorf("background module %q: %w", bg.Name(), err)
			}
		}(bg)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}

func (a *Application) setupSignalHandler(ctx context.Context, cancelFn context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case sig := <-sigChan:
		a.logger.Info("received signal", "signal", sig.String())
		cancelFn()
	case <-ctx.Done():
		return
	}
}

func (a *Application) runHooksBeforeStart(ctx context.Context) error {
	for _, h := range a.hooks {
		if h.BeforeStart != nil {
			if err := h.BeforeStart(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Application) runHooksAfterStart(ctx context.Context) error {
	for _, h := range a.hooks {
		if h.AfterStart != nil {
			if err := h.AfterStart(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Application) runHooksBeforeStop(ctx context.Context) error {
	for _, h := range a.hooks {
		if h.BeforeStop != nil {
			if err := h.BeforeStop(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Application) runHooksAfterStop(ctx context.Context) error {
	for _, h := range a.hooks {
		if h.AfterStop != nil {
			if err := h.AfterStop(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}
