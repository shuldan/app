package app

import "time"

func WithName(name string) func(application *Application) {
	return func(application *Application) {
		application.meta.name = name
	}
}

func WithVersion(version string) func(application *Application) {
	return func(application *Application) {
		application.meta.version = version
	}
}

func WithEnvironment(environment string) func(application *Application) {
	return func(application *Application) {
		application.meta.environment = environment
	}
}

func WithGracefulTimeout(timeout time.Duration) func(*Application) {
	return func(a *Application) {
		a.shutdownTimeout = timeout
	}
}
