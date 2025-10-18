# Go Observer Pattern

A lightweight, non-blocking generic implementation of the Observer Pattern for Go, inspired by Refactoring Guru’s design pattern documentation.

This module provides a thread-safe, event-driven pub/sub mechanism that allows multiple subscribers to asynchronously receive updates from a publisher without blocking event emission.

## Overview

The NonBlockingPublisher[T] type implements a generic, concurrent-safe publisher that distributes events of any type T to registered subscribers implementing the NonBlockingSubscriber[T] interface.

Each subscriber receives events via its own channel and defines a timeout threshold for handling messages, ensuring that a slow or blocked subscriber won’t stall the entire event pipeline.

This makes it ideal for use cases where event throughput and concurrency are important, such as:
   
    -	System monitoring and notifications
    -	Asynchronous background job processing
    -	Event streaming within microservices
    -	Real-time analytics and telemetry pipelines

### Features
    -	Type-safe generics (any-typed publisher and subscriber)
    -	Non-blocking event dispatch using goroutines and timeouts
    -	Thread-safe subscriber management
    -   Graceful shutdown via DrainThenStop()
    -	Immediate halt via Halt()
    -	Structured logging using Go’s standard log/slog package
    -	Context-aware cancellation and timeout control per subscriber

## Installation
To install the package, run:

```bash
go get github.com/jeremyforan/go-observer-pattern
```

