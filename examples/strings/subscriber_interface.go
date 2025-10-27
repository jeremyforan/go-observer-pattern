package main

import "time"

type StringSubscriber struct {
	id string
	c  chan string
}

func (s *StringSubscriber) ID() string {
	return s.id
}

func (s *StringSubscriber) Channel() chan<- string {
	return s.c
}

func (s *StringSubscriber) TimeoutThreshold() time.Duration {
	return 1 * time.Second
}
