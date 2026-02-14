package app

import (
	"errors"
	"strings"
	"testing"
)

func TestRegistry_Register_Success(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	m := &mockModule{name: "mod1"}
	if err := r.register(m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	all := r.getAll()
	if len(all) != 1 {
		t.Errorf("expected 1 module, got %d", len(all))
	}
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	m := &mockModule{name: ""}
	err := r.register(m)
	if !errors.Is(err, ErrModuleNameEmpty) {
		t.Errorf("expected ErrModuleNameEmpty, got %v", err)
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	m := &mockModule{name: "mod1"}
	_ = r.register(m)
	err := r.register(&mockModule{name: "mod1"})
	if !errors.Is(err, ErrModuleAlreadyRegistered) {
		t.Errorf("expected ErrModuleAlreadyRegistered, got %v", err)
	}
	if !strings.Contains(err.Error(), "mod1") {
		t.Errorf("expected error to contain module name, got %v", err)
	}
}

func TestRegistry_Register_Locked(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	r.lock()
	err := r.register(&mockModule{name: "mod1"})
	if !errors.Is(err, ErrRegistrationClosed) {
		t.Errorf("expected ErrRegistrationClosed, got %v", err)
	}
}

func TestRegistry_GetAll_ReturnsCopy(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	_ = r.register(&mockModule{name: "a"})
	_ = r.register(&mockModule{name: "b"})
	all := r.getAll()
	if len(all) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(all))
	}
	all[0] = nil
	original := r.getAll()
	if original[0] == nil {
		t.Errorf("getAll should return a copy, but original was modified")
	}
}

func TestRegistry_GetAll_Empty(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	all := r.getAll()
	if len(all) != 0 {
		t.Errorf("expected 0 modules, got %d", len(all))
	}
}

func TestRegistry_Lock(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	if r.locked.Load() {
		t.Fatal("expected not locked initially")
	}
	r.lock()
	if !r.locked.Load() {
		t.Fatal("expected locked after lock()")
	}
}
