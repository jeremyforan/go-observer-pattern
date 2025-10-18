package go_observer_pattern

import "time"

// NonBlockingSubscriber defines the interface that a subscriber must implement to be used
// with a NonBlockingPublisher. It is a generic interface that can handle events/notices of any type T.
// The subscriber must provide a unique identifier, a channel to receive events, and a timeout
// threshold for processing events.
//
// The subscriber is responsible for providing a channel, and processing the events received on that channel.
// When the subscriber is no longer interested in receiving events, it should unregister itself from the publisher.
//
// Panics may occur if the subscriber closes it's channel before unregistering from the publisher.
type NonBlockingSubscriber[T any] interface {

	// GetID returns a unique identifier for the subscriber. This ID is used by the publisher
	// to manage subscribers (e.g., registering and unregistering). And it must be unique across all subscribers.
	// THe ID is used as the key in the publisher's internal map of subscribers.
	GetID() string

	// GetChannel is used to obtain the channel of the subscriber when an event/notice
	// needs to be delivered to that subscriber. The publisher will send events to this channel when
	// notifying subscribers. The subscriber is responsible for reading from this channel.
	GetChannel() chan<- T

	// GetTimeoutThreshold expects a time.duration which is used to create a context.WithTimeout for each event/notice.
	// This ensures that an event will eventually fail instead of blocking a thread indefinitely.
	// If the subscriber does not process the event within the timeout duration, the event/notice is cancelled.
	GetTimeoutThreshold() time.Duration
}
