package application

import (
	"context"
	"errors"
	"log/slog"
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
	isRunning       int32
	shutdownTimeout time.Duration
}

func New(opts ...func(*Application)) *Application {
	reg := &registry{
		modules: make([]Module, 0),
		mu:      sync.RWMutex{},
	}
	a := &Application{
		registry: reg,
		runner: &runner{
			registry: reg,
		},
		shutdownTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

func (a *Application) Register(module Module) error {
	return a.registry.register(module)
}

func (a *Application) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go a.setupSignalHandler(ctx, cancel)

	if err := a.start(ctx, cancel); err != nil {
		return err
	}

	<-ctx.Done()

	if a.shutdownTimeout > 0 {
		shutdownCtx, timeoutCancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer timeoutCancel()

		errCh := make(chan error, 1)
		go func() {
			if regErr := a.runner.shutdownAll(ctx); regErr != nil {
				errCh <- regErr
			} else {
				errCh <- nil
			}
		}()

		var err error
		select {
		case err = <-errCh:
		case <-shutdownCtx.Done():
			err = errors.New("graceful shutdownAll timed out after " + a.shutdownTimeout.String())
		}

		return err
	}

	if err := a.runner.shutdownAll(ctx); err != nil {
		return err
	}

	return nil
}

func (a *Application) start(ctx context.Context, cancelFn context.CancelFunc) error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 0, 1) {
		return errors.New("application is already running")
	}

	a.meta.startTime = time.Now()

	slog.Info(
		"application is starting",
		"time", a.meta.startTime,
		"name", a.meta.name,
		"version", a.meta.version,
		"environment", a.meta.environment,
	)

	if err := a.runner.startAll(ctx); err != nil {
		if err := a.stop(cancelFn); err != nil {
			return err
		}
		return err
	}

	slog.Info(
		"modules started",
		"time", a.meta.startTime,
		"name", a.meta.name,
		"version", a.meta.version,
		"environment", a.meta.environment,
	)

	return nil
}

func (a *Application) stop(cancelFn context.CancelFunc) error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 1, 0) {
		return errors.New("application is already stopped")
	}
	cancelFn()
	a.meta.stopTime = time.Now()

	slog.Info(
		"application is stopping",
		"time", a.meta.stopTime,
		"name", a.meta.name,
		"version", a.meta.version,
		"environment", a.meta.environment,
	)

	return nil
}

func (a *Application) setupSignalHandler(ctx context.Context, cancelFn context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-sigChan:
		slog.Info(
			"application is shutting down",
			"time", time.Now(),
			"name", a.meta.name,
			"version", a.meta.version,
			"environment", a.meta.environment,
		)
		_ = a.stop(cancelFn)
	case <-ctx.Done():
		slog.Info(
			"application is shutting down",
			"time", time.Now(),
			"name", a.meta.name,
			"version", a.meta.version,
			"environment", a.meta.environment,
		)
		return
	}
}
