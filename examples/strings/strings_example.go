package main

import (
	"github.com/jeremyforan/observer"
)

func main() {
	publisher := observer.NewNonBlockingPublisher[string]()
	defer publisher.DrainThenStop()

	eventChan := publisher.Start()

	subscriber1 := &StringSubscriber{id: "First Subscriber"}
	subscriber2 := &StringSubscriber{id: "Second Subscriber"}

	_ = publisher.AddSubscriber(subscriber1)
	_ = publisher.AddSubscriber(subscriber2)

	eventChan <- "Event 1"
	eventChan <- "Event 2"

	subscriber3 := &StringSubscriber{id: "Third Subscriber"}
	_ = publisher.AddSubscriber(subscriber3)

	eventChan <- "Event 3"
}
