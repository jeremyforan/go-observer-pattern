package observer

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

var ErrSubscriberExists = errors.New("subscriber already registered with this publisher")

// NonBlockingPublisher is a generic implementation of the observer pattern. It will notify all registered
// subscribers of new events. This is based on the documentation here:
// https://refactoring.guru/design-patterns/observer
type NonBlockingPublisher[T any] struct {

	// mutex to protect access to the subscribers map.
	mu sync.RWMutex

	// map of subscribers, keyed by their unique ID.
	subscribers map[string]NonBlockingSubscriber[T]

	// eventChan is the primary channel for which incoming events are received and then deliver
	// to each of the subscribers.
	eventChan chan T

	// cancel chan is created as a cancel context and passed down to the threads that are
	// creating to deliver events.
	cancel chan struct{}

	// WaitGroup to ensure all subscriber notifications are processed before accepting new events.
	wg sync.WaitGroup

	// WaitGroup used to ensure all events are processed before shutdown. Used in DrainThenStop.
	shutdownWg sync.WaitGroup

	// structured logger for logging events and errors.
	logger *slog.Logger
}

// NewNonBlockingPublisher creates a new Publisher instance.
func NewNonBlockingPublisher[T any]() *NonBlockingPublisher[T] {
	l := slog.New(slog.DiscardHandler)
	return &NonBlockingPublisher[T]{
		subscribers: make(map[string]NonBlockingSubscriber[T]),
		eventChan:   make(chan T),
		cancel:      make(chan struct{}),
		logger:      l,
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

// DrainThenStop closes the event channel and waits for all subscribers to finish processing.
func (d *NonBlockingPublisher[T]) DrainThenStop() {
	time.Sleep(10 * time.Millisecond)

	if d.eventChan != nil {
		close(d.eventChan)
	}

	d.wg.Wait()
	d.shutdownWg.Wait()
}

// Halt stops the publisher immediately. It closes the event channel and cancels any in-progress events.
func (d *NonBlockingPublisher[T]) Halt() {
	if d.eventChan != nil {
		close(d.eventChan)
	}

	d.Cancel()
}

// AddSubscriber registers a new subscriber to receive updates.
// It returns an error if a subscriber with the same ID.
func (d *NonBlockingPublisher[T]) AddSubscriber(sub NonBlockingSubscriber[T]) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := sub.ID()

	if _, exists := d.subscribers[id]; exists {
		return ErrSubscriberExists
	}

	d.subscribers[id] = sub
	return nil
}

// RemoveSubscriber removes a subscriber from the list of subscribers.
// If the subscriber is not found, it does nothing.
func (d *NonBlockingPublisher[T]) RemoveSubscriber(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.subscribers, id)
	d.logger.Info("unregistered subscriber", "id", id)
}

// NotifySubscribers notifies all registered subscribers of a new event.
func (d *NonBlockingPublisher[T]) NotifySubscribers(ctx context.Context, e T) {
	subscribersCopy := d.subscriberReadSafeCopy()

	if len(subscribersCopy) == 0 {
		d.logger.Warn("no subscribers to notify", "event", e)
		return
	}

	for i := range subscribersCopy {
		go func(sub NonBlockingSubscriber[T]) {
			d.wg.Go(func() {

				to, cancel := context.WithTimeout(ctx, sub.TimeoutThreshold())
				defer cancel()

				c := sub.Channel()
				select {
				case c <- e:
					d.logger.Info("-->", "id", sub.ID(), "event", e)

				// if the context times out or is cancelled, log a warning
				case <-to.Done():
					d.logger.Warn("", "id", sub.ID(), "event", e, "err", to.Err())
				}
			})
		}(subscribersCopy[i])
	}
}

// SubscriberCount returns the number of registered subscribers. Used primarily to set the initial capacity of the slice in
// subscriberReadSafeCopy.
func (d *NonBlockingPublisher[T]) SubscriberCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.subscribers)
}

// Cancel stops the publisher from processing any new events. This does not close the event channel, so any events
// already in the channel will still be processed.
func (d *NonBlockingPublisher[T]) Cancel() {
	d.cancel <- struct{}{}
}

// subscriberReadSafeCopy makes a copy of the current subscribers in a thread-safe manner.
func (d *NonBlockingPublisher[T]) subscriberReadSafeCopy() []NonBlockingSubscriber[T] {
	d.mu.RLock()
	defer d.mu.RUnlock()

	subscribersCopy := make([]NonBlockingSubscriber[T], 0, d.SubscriberCount())

	for id := range d.subscribers {
		subscribersCopy = append(subscribersCopy, d.subscribers[id])
	}

	return subscribersCopy
}

// SetLogger sets the structured logger for the publisher.
func (d *NonBlockingPublisher[T]) SetLogger(logger *slog.Logger) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger = logger
}

// Logger returns the structured logger for the publisher.
func (d *NonBlockingPublisher[T]) Logger() *slog.Logger {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.logger
}
