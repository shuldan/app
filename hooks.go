package app

import "context"

type Hook struct {
	BeforeStart func(ctx context.Context) error
	AfterStart  func(ctx context.Context) error
	BeforeStop  func(ctx context.Context) error
	AfterStop   func(ctx context.Context) error
}
