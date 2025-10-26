package application

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

type Application struct {
	name            string
	version         string
	environment     string
	ctx             context.Context
	cancel          context.CancelFunc
	startTime       time.Time
	stopTime        time.Time
	registry        registry
	isRunning       int32
	shutdownTimeout time.Duration
}

func New(opts ...func(*Application)) *Application {
	ctx, cancel := context.WithCancel(context.Background())
	a := &Application{
		ctx:    ctx,
		cancel: cancel,
		registry: registry{
			modules: make([]Module, 0),
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

func (a *Application) Run() error {
	go a.setupSignalHandler()

	if err := a.start(); err != nil {
		return err
	}

	<-a.ctx.Done()

	if a.shutdownTimeout > 0 {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()

		errCh := make(chan error, 1)
		go func() {
			if regErr := a.registry.shutdownAll(); regErr != nil {
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

	if err := a.registry.shutdownAll(); err != nil {
		return err
	}

	return nil
}

func (a *Application) start() error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 0, 1) {
		return errors.New("application is already running")
	}

	a.startTime = time.Now()

	slog.Info(
		"application is starting",
		"time", a.startTime,
		"name", a.name,
		"version", a.version,
		"environment", a.environment,
	)

	if err := a.registry.startAll(); err != nil {
		if err := a.stop(); err != nil {
			return err
		}
		return err
	}

	slog.Info(
		"modules started",
		"time", a.startTime,
		"name", a.name,
		"version", a.version,
		"environment", a.environment,
	)

	return nil
}

func (a *Application) stop() error {
	if !atomic.CompareAndSwapInt32(&a.isRunning, 1, 0) {
		return errors.New("application is already stopped")
	}
	a.cancel()
	a.stopTime = time.Now()

	slog.Info(
		"application is stopping",
		"time", a.stopTime,
		"name", a.name,
		"version", a.version,
		"environment", a.environment,
	)

	return nil
}

func (a *Application) setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-sigChan:
		slog.Info(
			"application is shutting down",
			"time", time.Now(),
			"name", a.name,
			"version", a.version,
			"environment", a.environment,
		)
		_ = a.stop()
	case <-a.ctx.Done():
		slog.Info(
			"application is shutting down",
			"time", time.Now(),
			"name", a.name,
			"version", a.version,
			"environment", a.environment,
		)
		return
	}
}
