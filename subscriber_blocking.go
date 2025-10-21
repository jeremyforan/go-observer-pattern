package observer

type Subscriber[T any] interface {
	ID() string
	ReceiveUpdate(T)
}
