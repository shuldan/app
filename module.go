package app

import "context"

type Module interface {
	Name() string
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type BackgroundModule interface {
	Module
	Err() <-chan error
}

type HealthChecker interface {
	Health(ctx context.Context) error
}
