package application

import "time"

func WithName(name string) func(application *Application) {
	return func(application *Application) {
		application.name = name
	}
}

func WithVersion(version string) func(application *Application) {
	return func(application *Application) {
		application.version = version
	}
}

func WithEnvironment(environment string) func(application *Application) {
	return func(application *Application) {
		application.environment = environment
	}
}

func WithGracefulTimeout(timeout time.Duration) func(*Application) {
	return func(a *Application) {
		a.shutdownTimeout = timeout
	}
}
