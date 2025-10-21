package main

import "fmt"

type StringSubscriber struct {
	id string
}

func (s *StringSubscriber) ID() string {
	return s.id
}

func (s *StringSubscriber) ReceiveUpdate(event string) {
	fmt.Printf("Subscriber %s received event: %s\n", s.id, event)
}
