package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jeremyforan/observer"
)

func main() {

	// Create a non-blocking publisher with string events
	publisher := observer.NewNonBlockingPublisher[UniqueNoticeType]()

	// Optionally, set a logger to capture publisher events. By default, slog.DiscardHandler is used.
	publisher.SetLogger(slog.Default())

	//Start returns the channel that can be used to publish events
	pc := publisher.Start()
	defer publisher.DrainThenStop()

	// Create some subscribers with different timeout thresholds
	s1 := NewSubscriber("John", 1*time.Second)
	s2 := NewSubscriber("Paul", 2*time.Second)
	s3 := NewSubscriber("George", 3*time.Second)

	// Register the subscribers with the publisher
	_ = publisher.AddSubscriber(s1)
	_ = publisher.AddSubscriber(s2)
	_ = publisher.AddSubscriber(s3)

	// If the publisher already has a subscriber with the same ID, an error is returned
	err := publisher.AddSubscriber(s3)
	if err != nil {
		fmt.Println("error:", err)
	}

	// Cant forget about Ringo!
	_ = publisher.AddSubscriber(NewSubscriber("Ringo", 2*time.Second))

	// Publish some events
	pc <- startNotice
	pc <- interNotice
	pc <- endNotice

	// Remove Paul from the subscriber list
	publisher.RemoveSubscriber(s2.ID())

	// Publish some more events
	pc <- startNotice
	pc <- interNotice
	pc <- interNotice
	pc <- endNotice

	// Because of the deferred DrainThenStop call above, the publisher will
	// wait for all subscribers to finish processing or timeout before exiting main.

}

type UniqueNoticeType int

const (
	startNotice UniqueNoticeType = iota + 1
	interNotice
	endNotice
)

type NoticeSubscriber struct {
	id string
	c  chan UniqueNoticeType
	t  time.Duration
}

func NewSubscriber(id string, t time.Duration) *NoticeSubscriber {
	s := NoticeSubscriber{
		id: id,
		c:  make(chan UniqueNoticeType), // Buffered channel to avoid blocking
		t:  t,
	}

	go func() {
		for event := range s.c {
			// business logic for handling an event/notice goes here
			fmt.Println("s:", s.id, "e:", event)
		}
	}()

	return &s
}

// The following methods satisfy the NonBlockingSubscriber[T] interface

func (s *NoticeSubscriber) ID() string {
	return s.id
}

func (s *NoticeSubscriber) Channel() chan<- UniqueNoticeType {
	return s.c
}

func (s *NoticeSubscriber) TimeoutThreshold() time.Duration {
	return s.t
}
