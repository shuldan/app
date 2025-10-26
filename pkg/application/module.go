package application

type Module interface {
	Register() error
	Start() error
	Stop() error
}
