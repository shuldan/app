package application

import (
	"context"
	"time"
)

type appNameKeyType struct{}
type appVersionKeyType struct{}
type appEnvironmentKeyType struct{}
type appStartTimeKeyType struct{}

var (
	ContextKeyAppName        = appNameKeyType{}
	ContextKeyAppVersion     = appVersionKeyType{}
	ContextKeyAppEnvironment = appEnvironmentKeyType{}
	ContextKeyAppStartTime   = appStartTimeKeyType{}
)

type meta struct {
	name        string
	version     string
	environment string
	startTime   time.Time
	stopTime    time.Time
}

func (m *meta) enrichContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, ContextKeyAppName, m.name)
	ctx = context.WithValue(ctx, ContextKeyAppVersion, m.version)
	ctx = context.WithValue(ctx, ContextKeyAppEnvironment, m.environment)
	ctx = context.WithValue(ctx, ContextKeyAppStartTime, m.startTime)
	return ctx
}
