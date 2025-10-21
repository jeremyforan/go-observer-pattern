package main

import (
	"github.com/jeremyforan/observer"
)

func main() {
	publisher := observer.NewPublisher[string]()
	defer publisher.Stop()

	eventChan := publisher.Start()

	subscriber1 := &StringSubscriber{id: "First Subscriber"}
	subscriber2 := &StringSubscriber{id: "Second Subscriber"}

	_ = publisher.RegisterSubscriber(subscriber1)
	_ = publisher.RegisterSubscriber(subscriber2)

	eventChan <- "Event 1"
	eventChan <- "Event 2"

	subscriber3 := &StringSubscriber{id: "Third Subscriber"}
	_ = publisher.RegisterSubscriber(subscriber3)

	eventChan <- "Event 3"
}
