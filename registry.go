package app

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type registry struct {
	modules []Module
	names   map[string]struct{}
	mu      sync.RWMutex
	locked  atomic.Bool
}

func newRegistry() *registry {
	return &registry{
		modules: make([]Module, 0),
		names:   make(map[string]struct{}),
	}
}

func (r *registry) register(module Module) error {
	if r.locked.Load() {
		return ErrRegistrationClosed
	}

	name := module.Name()
	if name == "" {
		return ErrModuleNameEmpty
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.names[name]; exists {
		return fmt.Errorf("%w: %s", ErrModuleAlreadyRegistered, name)
	}

	r.names[name] = struct{}{}
	r.modules = append(r.modules, module)
	return nil
}

func (r *registry) lock() {
	r.locked.Store(true)
}

func (r *registry) getAll() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Module, len(r.modules))
	copy(result, r.modules)
	return result
}
