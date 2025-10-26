package application

import (
	"errors"
	"sync"
)

type registry struct {
	modules []Module
	mu      sync.RWMutex
}

func (r *registry) register(module Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules = append(r.modules, module)
	return nil
}

func (r *registry) all() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Module, len(r.modules))
	copy(result, r.modules)
	return result
}

func (r *registry) startAll() error {
	for _, module := range r.all() {
		if err := module.Register(); err != nil {
			return err
		}
	}
	started := 0
	for _, module := range r.all() {
		if err := module.Start(); err != nil {
			r.shutdownStarted(started)
			return err
		}
		started++
	}

	return nil
}

func (r *registry) shutdownStarted(startedModulesCount int) {
	modules := r.all()
	startedModulesCount = min(len(modules), startedModulesCount)
	for i := startedModulesCount - 1; i >= 0; i-- {
		_ = modules[i].Stop()
	}
}

func (r *registry) shutdownAll() error {
	var errs []error
	modules := r.all()
	for i := len(modules) - 1; i >= 0; i-- {
		if err := modules[i].Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
