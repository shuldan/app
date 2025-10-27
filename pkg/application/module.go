package application

import "context"

type Module interface {
	Register(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
