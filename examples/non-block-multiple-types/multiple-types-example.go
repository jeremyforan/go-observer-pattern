package main

import (
	"fmt"
	"go-observer-pattern"
	"log/slog"
	"time"
)

func main() {

	// Create a non-blocking publisher with string events
	publisher := go_observer_pattern.NewNonBlockingPublisher[Event]()

	// Optionally, set a logger to capture publisher events. By default, slog.DiscardHandler is used.
	publisher.SetLogger(slog.Default())

	//Start returns the channel that can be used to publish events
	pc := publisher.Start()
	defer publisher.DrainThenStop()

	// Create a subscriber that handles multiple event types
	s1 := NewSubscriber("John", 1*time.Second, func(e Event) {
		switch ev := e.(type) {
		case UpdateEvent:
			fmt.Printf("Subscriber %s received UpdateEvent: %s\n", "John", ev.Details)
		case EventUpgrade:
			fmt.Printf("Subscriber %s received EventUpgrade: %s\n", "John", ev.Version)
		default:
			fmt.Printf("Subscriber %s received unknown event type\n", "John")
		}
	})

	// Create another subscriber that only handles UpdateEvent
	s2 := NewSubscriber("Paul", 2*time.Second, func(e Event) {
		if ue, ok := e.(UpdateEvent); ok {
			fmt.Printf("Subscriber %s received UpdateEvent: %s\n", "Paul", ue.Details)
		}
	})

	// Create a third subscriber that only handles EventUpgrade
	s3 := NewSubscriber("George", 3*time.Second, func(e Event) {
		if eu, ok := e.(EventUpgrade); ok {
			fmt.Printf("Subscriber %s received EventUpgrade: %s\n", "George", eu.Version)
		}
	})

	// Register the subscribers with the publisher
	_ = publisher.AddSubscriber(s1)
	_ = publisher.AddSubscriber(s2)
	_ = publisher.AddSubscriber(s3)

	// If the publisher already has a subscriber with the same ID, an error is returned
	err := publisher.AddSubscriber(s3)
	if err != nil {
		fmt.Println("error:", err)
	}

	// Publish some events
	pc <- UpdateEvent{Details: "New Blog Post Available"}
	pc <- EventUpgrade{Version: "v2.0.1"}

	// Remove Paul from the subscriber list
	publisher.RemoveSubscriber(s2.GetID())

	// Publish some more events
	pc <- UpdateEvent{Details: "System Maintenance Scheduled"}

	// Because of the deferred DrainThenStop call above, the publisher will
	// wait for all subscribers to finish processing or timeout before exiting main.

}

type Event interface {
	EventType() EventType
}

type EventType string

const (
	eventUpdate  EventType = "UPDATE"
	eventUpgrade EventType = "UPGRADE"
)

type UpdateEvent struct {
	Details string
}

func (e UpdateEvent) EventType() EventType {
	return eventUpdate
}

type EventUpgrade struct {
	Version string
}

func (e EventUpgrade) EventType() EventType {
	return eventUpgrade
}

type NoticeSubscriber struct {
	id string
	c  chan Event
	t  time.Duration
}

func NewSubscriber(id string, t time.Duration, f func(Event)) *NoticeSubscriber {
	s := NoticeSubscriber{
		id: id,
		c:  make(chan Event), // Buffered channel to avoid blocking
		t:  t,
	}

	go func() {
		for event := range s.c {
			// business logic for handling an event/notice goes here
			f(event)
		}
	}()

	return &s
}

// The following methods satisfy the NonBlockingSubscriber[T] interface

func (s *NoticeSubscriber) GetID() string {
	return s.id
}

func (s *NoticeSubscriber) GetChannel() chan<- Event {
	return s.c
}

func (s *NoticeSubscriber) GetTimeoutThreshold() time.Duration {
	return s.t
}
