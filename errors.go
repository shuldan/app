package app

import "errors"

var (
	ErrApplicationAlreadyRunning  = errors.New("application is already running")
	ErrApplicationAlreadyStopped  = errors.New("application is already stopped")
	ErrGracefulShutdownTimedOut   = errors.New("graceful shutdown timed out")
	ErrRegistrationClosed         = errors.New("registration is closed: application already started")
	ErrModuleAlreadyRegistered    = errors.New("module already registered")
	ErrModuleNameEmpty            = errors.New("module name must not be empty")
	ErrAppNameEmpty               = errors.New("application name must not be empty")
	ErrShutdownTimeoutNonPositive = errors.New("shutdown timeout must be positive or zero")
)
