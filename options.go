package app

import "time"

type Option func(*Application) error

func WithName(name string) Option {
	return func(a *Application) error {
		if name == "" {
			return ErrAppNameEmpty
		}
		a.meta.name = name
		return nil
	}
}

func WithVersion(version string) Option {
	return func(a *Application) error {
		a.meta.version = version
		return nil
	}
}

func WithEnvironment(environment string) Option {
	return func(a *Application) error {
		a.meta.environment = environment
		return nil
	}
}

func WithGracefulTimeout(timeout time.Duration) Option {
	return func(a *Application) error {
		if timeout < 0 {
			return ErrShutdownTimeoutNonPositive
		}
		a.shutdownTimeout = timeout
		return nil
	}
}

func WithLogger(logger Logger) Option {
	return func(a *Application) error {
		if logger != nil {
			a.logger = logger
		}
		return nil
	}
}

func WithHook(hook Hook) Option {
	return func(a *Application) error {
		a.hooks = append(a.hooks, hook)
		return nil
	}
}
