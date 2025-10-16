package main

import (
	"fmt"
	"go-observer-pattern"
	"time"
)

func main() {

	publisher := go_observer_pattern.NewNonBlockingPublisher[string]()

	eventChan := publisher.Start()
	defer publisher.DrainThenStop()

	subscriber1 := NewStringSubscriber("Fizz", 1*time.Second)
	subscriber1.StartListening(func(e string) {
		fmt.Println("sub 1 completed event:", e)
	})

	subscriber2 := NewStringSubscriber("Buzz", 2*time.Second)
	subscriber2.StartListening(func(e string) {
		time.Sleep(5 * time.Second)
		fmt.Println("sub 2 - completed event:", e)
	})

	subscriber3 := NewStringSubscriber("Pop", 2*time.Second)
	subscriber3.StartListening(func(e string) {
		time.Sleep(1 * time.Second)
		fmt.Println("sub 3 - completed event:", e)
	})

	_ = publisher.RegisterSubscriber(subscriber1)
	_ = publisher.RegisterSubscriber(subscriber2)

	err := publisher.RegisterSubscriber(subscriber3)
	if err != nil {
		fmt.Println("error:", err)
	}

	eventChan <- "Breakfast"
	eventChan <- "Lunch"
	eventChan <- "Dinner"
	eventChan <- "Supper"

	time.Sleep(20 * time.Millisecond)

}
