package client

import (
	"github.com/easymicro/log"
	"github.com/sony/gobreaker"
)

func NewReadyToTrip() func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		return true
	}
}

func NewOnStateChange() func(name string, from gobreaker.State, to gobreaker.State) {
	f := func(state gobreaker.State) string {
		switch state {
		case gobreaker.StateClosed:
			return "close"
		case gobreaker.StateHalfOpen:
			return "halfOpen"
		case gobreaker.StateOpen:
			return "open"
		}
		return ""
	}
	return func(name string, from gobreaker.State, to gobreaker.State) {
		log.Errorf("name %s change from %s to %s", name, f(from), f(to))
	}
}
