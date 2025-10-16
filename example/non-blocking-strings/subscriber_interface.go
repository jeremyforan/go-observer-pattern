package main

import (
	"time"
)

type StringSubscriber struct {
	id string
	c  chan string
	t  time.Duration
}

func NewStringSubscriber(id string, t time.Duration) *StringSubscriber {
	return &StringSubscriber{
		id: id,
		c:  make(chan string), // Buffered channel to avoid blocking
		t:  t,
	}
}

func (s *StringSubscriber) GetId() string {
	return s.id
}

func (s *StringSubscriber) GetChannel() chan<- string {
	return s.c
}

func (s *StringSubscriber) StartListening(fun func(string)) {
	go func(fu func(string)) {
		for event := range s.c {
			fu(event)
		}
	}(fun)
}

func (s *StringSubscriber) GetTimeout() time.Duration {
	return s.t
}
