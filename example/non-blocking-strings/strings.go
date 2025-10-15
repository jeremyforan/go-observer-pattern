package main

import (
	"fmt"
	"go-observer-pattern"
	"time"
)

func publish(event string) {
	fmt.Println("completed event:", event)
}

func main() {

	publisher := go_observer_pattern.NewNonBlockingPublisher[string]()

	eventChan := publisher.Start()

	subscriber1 := NewStringSubscriber("A", 1*time.Second)
	subscriber1.StartListening(publish)

	subscriber2 := NewStringSubscriber("B", 2*time.Second)
	subscriber2.StartListening(func(e string) {
		time.Sleep(5 * time.Second)
		publish(e)
	})

	_ = publisher.RegisterSubscriber(subscriber1)
	err := publisher.RegisterSubscriber(subscriber1)
	if err != nil {
		fmt.Println("error:", err)
	}
	_ = publisher.RegisterSubscriber(subscriber2)

	eventChan <- "1"
	eventChan <- "2"
	eventChan <- "3"
	eventChan <- "4"

	time.Sleep(20 * time.Millisecond)

	publisher.DrainAndStop()
}
