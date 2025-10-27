package application

import (
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

func (r *registry) getAll() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Module, len(r.modules))
	copy(result, r.modules)
	return result
}
