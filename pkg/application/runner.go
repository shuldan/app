package application

import (
	"context"
	"errors"
)

type runner struct {
	registry *registry
}

func (r *runner) startAll(ctx context.Context) error {
	for _, module := range r.registry.getAll() {
		if err := module.Register(ctx); err != nil {
			return err
		}
	}
	started := 0
	for _, module := range r.registry.getAll() {
		if err := module.Start(ctx); err != nil {
			r.shutdownStarted(ctx, started)
			return err
		}
		started++
	}

	return nil
}

func (r *runner) shutdownStarted(ctx context.Context, startedModulesCount int) {
	modules := r.registry.getAll()
	startedModulesCount = min(len(modules), startedModulesCount)
	for i := startedModulesCount - 1; i >= 0; i-- {
		_ = modules[i].Stop(ctx)
	}
}

func (r *runner) shutdownAll(ctx context.Context) error {
	var errs []error
	modules := r.registry.getAll()
	for i := len(modules) - 1; i >= 0; i-- {
		if err := modules[i].Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
