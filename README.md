 <h1 align="center">Go Observer Pattern</h1>
 <p align="center">A lightweight, non-blocking generic implementation of the Observer Pattern for Go, inspired by <a href="https://refactoring.guru/design-patterns/observer">Refactoring Guru’s design pattern.</a>
</p>
<p align="center">
	<a href="https://goreportcard.com/badge/github.com/jeremyforan/go-observer-pattern"><img src="https://goreportcard.com/badge/github.com/jeremyforan/go-observer-pattern" /></a>
	<img alt="GitHub License" src="https://img.shields.io/github/license/jeremyforan/observer">
	<img alt="GitHub go.mod Go version" src="https://img.shields.io/github/go-mod/go-version/jeremyforan/observer">
	<img alt="GitHub Tag" src="https://img.shields.io/github/v/tag/jeremyforan/observer">
	<img alt="GitHub Actions Workflow Status" src="https://img.shields.io/github/actions/workflow/status/jeremyforan/observer/go.yml">
</p>

--- 

A lightweight, non-blocking generic implementation of the Observer Pattern for Go, inspired by [Refactoring Guru’s design pattern](https://refactoring.guru/design-patterns/observer).

This package provides a thread-safe, event-driven pub/sub mechanism that allows multiple subscribers to asynchronously receive updates from a publisher without blocking event emission.

Here’s the **Markdown anchor-linked Table of Contents** version — ready to drop right into your `README.md` on GitHub:

---

## Table of Contents

 - [Overview](#overview)

   * [Features](#features)
 - [Installation](#installation)
 - [Full Example](#full-example)
 - [Subscriber Interface](#subscriber-interface)
 - [Publisher Usage](#publisher-usage)
   
   * [Adding and Removing Subscribers](#adding-and-removeing-subscribers)
   * [Publishing Events](#publishing-events)
 - [Logging](#logging)
 - [Error Handling](#error-handling)
 - [On-going Development](#on-going-development)

---

✅ **Tip:** GitHub automatically generates these anchor links based on the section headers (case-insensitive, spaces replaced with hyphens).
You can place this immediately after your project badges and before the “Overview” section for best readability.


## Overview

The **NonBlockingPublisher[T]** type implements a generic, concurrent-safe publisher that distributes events of any **type T** to registered subscribers implementing the **NonBlockingSubscriber[T]** interface.

Each subscriber receives events via its own channel and defines a timeout threshold for handling messages, ensuring that a slow or blocked subscriber won’t stall the entire event pipeline.

This makes it ideal for use cases where event throughput and concurrency are important, such as:
 - System monitoring and notifications 
 - Asynchronous background job processing 
 - Event streaming within microservices 
 - Real-time analytics and telemetry pipelines


### Features
- Type-safe generics (any-typed publisher and subscriber)
- Non-blocking event dispatch using goroutines and timeouts 
- Thread-safe subscriber management 
- Graceful shutdown via DrainThenStop()
- Structured logging using Go’s standard log/slog package 
- Context-aware cancellation and timeout control per subscriber 
- Zero external dependencies

## Installation
To install the package, run:

```bash
go get github.com/jeremyforan/go-observer-pattern
```

# Full Example
```go
package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jeremyforan/observer"
)

func main() {

	// Create a non-blocking publisher with UniqueNoticeType as the event type
	publisher := observer.NewNonBlockingPublisher[UniqueNoticeType]()

	// Optionally, set a logger to capture publisher events. 
	// By default, slog.DiscardHandler is used.
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
```

## Subscriber Interface

Any subscriber must implement the NonBlockingSubscriber[T] interface. The interface is simple to implement:

**ID() string** must return a unique identifier for the subscriber. This ID is used by the publisher
to manage subscribers (e.g., registering and unregistering). And it must be unique across all subscribers.
The ID is used as the key in the publisher's internal map of subscribers.

```go
func (s *MySubscriber) ID() string {
    return s.id
}
```

**Channel() chan<- T** is used to obtain the channel of the subscriber when an event/notice
needs to be delivered to that subscriber. The publisher will send events to this channel when
notifying subscribers. The subscriber is responsible for reading from this channel.

```go
func (s *MySubscriber) Channel() chan<- MyEventType {
    return s.channel
}
```

**TimeoutThreshold() time.Duration** expects a time.duration which is used to create a context.WithTimeout for each event/notice.
This ensures that an event will eventually fail instead of blocking a thread indefinitely.
If the subscriber does not process the event within the timeout duration, the event/notice is cancelled.

```go
func (s *MySubscriber) TimeoutThreshold() time.Duration {
    return s.timeoutDuration
}
```

A basic example implementation of a subscriber could look like this:

```go
type MySubscriber struct {
    id              string
    channel         chan<- MyEventType
    timeoutDuration time.Duration
}   
 
func (s *MySubscriber) ID() string {
    return s.id
}
func (s *MySubscriber) Channel() chan<- MyEventType {
    return s.channel
}
func (s *MySubscriber) TimeoutThreshold() time.Duration {
    return s.timeoutDuration
}
```

When the publisher sends an event, it will use the subscriber's channel to deliver the event.
The subscriber must read from this channel to receive events.   

You may implement any logic you want in the subscriber to handle the received events.

```go
func (s *MySubscriber) StartListening() {
    go func() {
        for event := range s.channel {
            // Process the event
            fmt.Printf("Subscriber %s received event: %v\n", s.id, event)
        }
    }()
}
```

## Publisher Usage
To use the NonBlockingPublisher[T], first create an instance of it:

```go
// example of a publisher that accepts string as the type
publisher := NewNonBlockingPublisher[string]()
```

### Adding and removeing subscribers

Then, create and adding subscribers:

```go
publisher.AddSubscriber(s1)
```
removing a subscriber is just as easy:

```go
publisher.RemoveSubscriber(s1.ID())
```

### Publishing events
Starting the publisher will return a channel that is used to publish the events:

```go
eventChannel := publisher.Start()
```

Anything that sends events to `eventChannel` will be published to all registered subscribers:

```go
eventChannel <- "let everyone know"
```

When you are done publishing events, you can stop the publisher gracefully:

```go
publisher.DrainThenStop()
```

or halt it immediately which abandons any pending events:

```go
publisher.Halt()
```

## Logging
This module uses Go's standard `log/slog` package for structured logging. By default the logger is set to the Discard handler, which means no logs will be output.
You can customize the logger by using the `SetLogger` method on the publisher:

```go
opts := &slog.HandlerOptions{
	Level: slog.LevelDebug
}

h := slog.NewTextHandler(os.Stdout, opts)
l := slog.New(h)

publisher.SetLogger(l)
```

This allows you to integrate the publisher's logging with your application's logging strategy.

### Error Handling

There is currently only one type of error returned by the publisher methods: `ErrSubscriberExists`.
This error is returned when attempting to add a subscriber with an ID that already exists in the publisher's subscriber list. 
Because subscriber IDs must be unique, this error helps prevent duplicate registrations.

```go
err := publisher.AddSubscriber(someSubscriber)
if errors.Is(err, go_observer_pattern.ErrSubscriberExists) {
    fmt.Println("subscriber already exists")
    return
}
```

## On-going Development

I am on the fence about removing the blocking version of the publisher since this non-blocking version is more versatile.
Feel free to open an issue or a pull request if you have suggestions or improvements!

