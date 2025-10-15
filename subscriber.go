package go_observer_pattern

type Subscriber[T any] interface {
	GetId() string
	ReceiveUpdate(T)
}
