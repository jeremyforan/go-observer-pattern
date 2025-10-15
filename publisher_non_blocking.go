package go_observer_pattern

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

var ErrSubscriberExists = errors.New("subscriber already registered with this publisher")

// NonBlockingPublisher is a generic implementation of the observer pattern. It will notify all registered subscribers of
// new events. This is based on the documentation here:
// https://refactoring.guru/design-patterns/observer
type NonBlockingPublisher[T any] struct {

	// map of subscribers, keyed by their unique ID.
	subscribers map[string]NonBlockingSubscriber[T]

	// channel for incoming events to be published to subscribers.
	eventChan chan T

	// channel to signal cancellation of event processing.
	cancel chan struct{}

	// wait group to ensure all subscriber notifications are processed before accepting new events.
	wg sync.WaitGroup

	// wait group to ensure all events are processed before shutdown. Used in DrainAndStop.
	shutdownWg sync.WaitGroup

	// mutex to protect access to the subscribers map.
	mu sync.RWMutex
}

// NewNonBlockingPublisher creates a new Publisher instance.
func NewNonBlockingPublisher[T any]() *NonBlockingPublisher[T] {
	return &NonBlockingPublisher[T]{
		subscribers: make(map[string]NonBlockingSubscriber[T]),
		eventChan:   make(chan T),
		cancel:      make(chan struct{}),
	}
}

// Start returns a *send-only* channel so callers can publish but not read. It will loop and notify subscribers of
// new events until the d.eventChan channel is closed.
func (d *NonBlockingPublisher[T]) Start() chan<- T {
	ctx, cancel := context.WithCancel(context.Background())

	// Listen for cancel signal to stop processing the current running events.
	go func() {
		<-d.cancel
		cancel()
	}()

	// add one to the shutdown wait group, this allows the go routine to 'drain' the event channel before exiting.
	d.shutdownWg.Add(1)
	go func() {
		defer d.shutdownWg.Done()
		for event := range d.eventChan {
			d.NotifySubscribers(ctx, event)
		}
	}()

	// implicitly converts to send-only
	return d.eventChan
}

// DrainAndStop closes the event channel and waits for all subscribers to finish processing.
func (d *NonBlockingPublisher[T]) DrainAndStop() {
	close(d.eventChan)
	d.wg.Wait()
	d.shutdownWg.Wait()
}

// RegisterSubscriber registers a new subscriber to receive updates.
// It returns an error if a subscriber with the same ID.
func (d *NonBlockingPublisher[T]) RegisterSubscriber(obs NonBlockingSubscriber[T]) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := obs.GetId()

	if _, exists := d.subscribers[id]; exists {
		return ErrSubscriberExists
	}

	d.subscribers[id] = obs
	return nil
}

// UnregisterSubscriber removes a subscriber from the list of subscribers.
// If the subscriber is not found, it does nothing.
func (d *NonBlockingPublisher[T]) UnregisterSubscriber(obs Subscriber[T]) {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := obs.GetId()
	delete(d.subscribers, id)
}

// NotifySubscribers notifies all registered subscribers of a new event.
func (d *NonBlockingPublisher[T]) NotifySubscribers(ctx context.Context, e T) {
	subscribersCopy := d.subscriberReadCopy()

	for i := range subscribersCopy {
		go func(_ctx context.Context, o NonBlockingSubscriber[T]) {
			d.wg.Go(func() {

				to, cancel := context.WithTimeout(_ctx, o.GetTimeout())
				defer cancel()

				c := o.GetChannel()
				select {
				case c <- e:
					slog.Info("dlvrd", "id", o.GetId(), "event", e)

				case <-to.Done():
					slog.Warn("ntdlr", "id", o.GetId(), "event", e, "err", to.Err())
				}
			})
		}(ctx, subscribersCopy[i])
	}
}

// SubscriberCount returns the number of registered subscribers. Used primarily to set the initial capacity of the slice in
// subscriberReadCopy.
func (d *NonBlockingPublisher[T]) SubscriberCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.subscribers)
}

// subscriberReadCopy makes a copy of the current subscribers in a thread-safe manner.
func (d *NonBlockingPublisher[T]) subscriberReadCopy() []NonBlockingSubscriber[T] {
	d.mu.RLock()
	defer d.mu.RUnlock()

	subscribersCopy := make([]NonBlockingSubscriber[T], 0, d.SubscriberCount())

	for id := range d.subscribers {
		subscribersCopy = append(subscribersCopy, d.subscribers[id])
	}

	return subscribersCopy
}

// Cancel stops the publisher from processing any new events. This does not close the event channel, so any events
// already in the channel will still be processed.
func (d *NonBlockingPublisher[T]) Cancel() {
	d.cancel <- struct{}{}
}

// Halt stops the publisher immediately. It closes the event channel and cancels any in-progress events.
func (d *NonBlockingPublisher[T]) Halt() {
	close(d.eventChan)
	d.Cancel()
}
