package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	ErrApplicationAlreadyRunning   = errors.New("application is already running")
	ErrApplicationAlreadyStopped   = errors.New("application is already stopped")
	ErrGracefulShutdownAllTimedOut = errors.New("graceful shutdownAll timed out")
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

		select {
		case err := <-errCh:
			return err
		case <-shutdownCtx.Done():
			return ErrGracefulShutdownAllTimedOut
		}
	}

	if err := a.runner.shutdownAll(ctx); err != nil {
		return err
	}

	return nil
}

func (a *Application) start(ctx context.Context, cancelFn context.CancelFunc) error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 0, 1) {
		return ErrApplicationAlreadyRunning
	}

	a.meta.startTime = time.Now()

	ctx = a.meta.enrichContext(ctx)

	if err := a.runner.startAll(ctx); err != nil {
		if err := a.stop(cancelFn); err != nil {
			return err
		}
		return err
	}

	return nil
}

func (a *Application) stop(cancelFn context.CancelFunc) error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 1, 0) {
		return ErrApplicationAlreadyStopped
	}
	cancelFn()
	a.meta.stopTime = time.Now()

	return nil
}

func (a *Application) setupSignalHandler(ctx context.Context, cancelFn context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-sigChan:
		_ = a.stop(cancelFn)
	case <-ctx.Done():
		return
	}
}
