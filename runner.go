package app

import (
	"context"
	"errors"
	"fmt"
)

type runner struct {
	registry *registry
	logger   Logger
}

func (r *runner) initAll(ctx context.Context) error {
	for _, module := range r.registry.getAll() {
		r.logger.Info("initializing module", "module", module.Name())
		if err := module.Init(ctx); err != nil {
			return fmt.Errorf("init module %q: %w", module.Name(), err)
		}
	}
	return nil
}

func (r *runner) startAll(ctx context.Context) (startedModules []Module, err error) {
	modules := r.registry.getAll()
	started := make([]Module, 0, len(modules))

	for _, module := range modules {
		r.logger.Info("starting module", "module", module.Name())
		if err := module.Start(ctx); err != nil {
			shutdownErr := r.shutdownModules(context.Background(), started)
			return nil, errors.Join(
				fmt.Errorf("start module %q: %w", module.Name(), err),
				shutdownErr,
			)
		}
		started = append(started, module)
	}

	return started, nil
}

func (r *runner) shutdownModules(ctx context.Context, modules []Module) error {
	var errs []error
	for i := len(modules) - 1; i >= 0; i-- {
		m := modules[i]
		r.logger.Info("stopping module", "module", m.Name())
		if err := m.Stop(ctx); err != nil {
			wrappedErr := fmt.Errorf("stop module %q: %w", m.Name(), err)
			r.logger.Error("failed to stop module", "module", m.Name(), "error", err)
			errs = append(errs, wrappedErr)
		}
	}
	return errors.Join(errs...)
}

func (r *runner) shutdownAll(ctx context.Context) error {
	return r.shutdownModules(ctx, r.registry.getAll())
}
