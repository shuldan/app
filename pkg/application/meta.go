package application

import "time"

type meta struct {
	name        string
	version     string
	environment string
	startTime   time.Time
	stopTime    time.Time
}
