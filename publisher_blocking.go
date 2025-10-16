package go_observer_pattern

import (
	"errors"
	"sync"
)

// Publisher is a generic implementation of the observer pattern. It will notify all registered subscribers of
// new events. This is based on the documentation here:
// https://refactoring.guru/design-patterns/observer
type Publisher[T any] struct {
	subscribers map[string]Subscriber[T]
	eventChan   chan T
	wg          sync.WaitGroup
	shutdownWg  sync.WaitGroup
	mu          sync.RWMutex
}

// NewPublisher creates a new Publisher instance.
func NewPublisher[T any]() *Publisher[T] {
	return &Publisher[T]{
		subscribers: make(map[string]Subscriber[T]),
		eventChan:   make(chan T),
	}
}

// NewPublisherWithBuffer creates a new Publisher instance with a buffered channel.
func NewPublisherWithBuffer[T any](bufferSize int) *Publisher[T] {
	return &Publisher[T]{
		subscribers: make(map[string]Subscriber[T]),
		eventChan:   make(chan T, bufferSize),
	}
}

// Start returns a *send-only* channel so callers can publish but not read. It will loop and notify subscribers of
// new events until the d.eventChan channel is closed.
func (d *Publisher[T]) Start() chan<- T {
	d.shutdownWg.Add(1)
	go func() {
		defer d.shutdownWg.Done()
		for event := range d.eventChan {
			d.NotifySubscribers(event)
		}
	}()
	return d.eventChan // implicitly converts to send-only
}

// Stop closes the event channel and waits for all subscribers to finish processing.
func (d *Publisher[T]) Stop() {
	close(d.eventChan)
	d.shutdownWg.Wait()
}

// RegisterSubscriber registers a new subscriber to receive updates.
// It returns an error if a subscriber with the same ID.
func (d *Publisher[T]) RegisterSubscriber(obs Subscriber[T]) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := obs.GetId()
	if _, exists := d.subscribers[id]; exists {
		return errors.New("observer with this ID already registered")
	}
	d.subscribers[id] = obs
	return nil
}

// UnregisterSubscriber removes a subscriber from the list of subscribers.
// If the subscriber is not found, it does nothing.
func (d *Publisher[T]) UnregisterSubscriber(obs Subscriber[T]) {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := obs.GetId()
	delete(d.subscribers, id)
}

// NotifySubscribers notifies all registered subscribers of a new event.
func (d *Publisher[T]) NotifySubscribers(s T) {
	d.mu.RLock()

	subscribersCopy := make([]Subscriber[T], 0, len(d.subscribers))
	for id := range d.subscribers {
		subscribersCopy = append(subscribersCopy, d.subscribers[id])
	}

	d.mu.RUnlock()

	d.wg.Add(len(subscribersCopy))

	for i := range subscribersCopy {
		go func(o Subscriber[T]) {
			defer d.wg.Done()
			o.ReceiveUpdate(s)
		}(subscribersCopy[i])
	}
	d.wg.Wait()
}
