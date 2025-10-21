package observer

type Subscriber[T any] interface {
	GetID() string
	ReceiveUpdate(T)
}
