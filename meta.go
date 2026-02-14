package app

import (
	"context"
	"time"
)

type appNameKeyType struct{}
type appVersionKeyType struct{}
type appEnvironmentKeyType struct{}
type appStartTimeKeyType struct{}

var (
	contextKeyAppName        = appNameKeyType{}
	contextKeyAppVersion     = appVersionKeyType{}
	contextKeyAppEnvironment = appEnvironmentKeyType{}
	contextKeyAppStartTime   = appStartTimeKeyType{}
)

type meta struct {
	name        string
	version     string
	environment string
	startTime   time.Time
	stopTime    time.Time
}

func (m *meta) enrichContext(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, contextKeyAppName, m.name)
	ctx = context.WithValue(ctx, contextKeyAppVersion, m.version)
	ctx = context.WithValue(ctx, contextKeyAppEnvironment, m.environment)
	ctx = context.WithValue(ctx, contextKeyAppStartTime, m.startTime)
	return ctx
}

func (m *meta) uptime() time.Duration {
	end := m.stopTime
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(m.startTime)
}

func NameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyAppName).(string)
	return v
}

func VersionFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyAppVersion).(string)
	return v
}

func EnvironmentFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyAppEnvironment).(string)
	return v
}

func StartTimeFromContext(ctx context.Context) time.Time {
	v, _ := ctx.Value(contextKeyAppStartTime).(time.Time)
	return v
}
