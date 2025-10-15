package go_observer_pattern

import "time"

type NonBlockingSubscriber[T any] interface {
	GetId() string
	GetChannel() chan<- T
	GetTimeout() time.Duration
}
