package observer

import (
	"log/slog"
	"testing"
	"testing/synctest"
	"time"
)

func TestNewNonBlockingPublisher(t *testing.T) {
	pub := NewNonBlockingPublisher[string]()

	t.Run("CreatesInstance", func(t *testing.T) {
		if pub == nil {
			t.Error("Expected publisher instance, got nil")
			t.Fail()
		}
	})

	s1 := NewSubscriberMock("sub1", 1*time.Second)
	t.Run("AddSubscriber", func(t *testing.T) {
		err := pub.AddSubscriber(s1)
		if err != nil {
			t.Errorf("Unexpected error adding subscriber: %v", err)
		}
	})

	err := pub.AddSubscriber(s1)
	t.Run("AttemptToAddExistingSubscriber", func(t *testing.T) {
		if err == nil {
			t.Error("Expected error when adding duplicate subscriber, got nil")
		}
	})

	t.Run("AttemptToAddSubscriberWithSameId", func(t *testing.T) {
		sameIDSub := NewSubscriberMock("sub1", 2*time.Second)
		err = pub.AddSubscriber(sameIDSub)
		if err == nil {
			t.Error("Expected error when adding subscriber with same ID, got nil")
		}
	})

	t.Run("RemoveSubscriber", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {

			const deliver = "event to deliver"
			//_ = pub.Start()

			//
			//received := make(chan bool)
			//s1.SetHandler(func(s string) {
			//	if s == deliver {
			//		received <- true
			//	}
			//})
			//
			//// send event before removal
			//ec <- deliver
			//select {
			//case <-received:
			//	// success
			//case <-time.After(2 * time.Second):
			//	t.Error("Subscriber did not receive event")
			//}
			//
			//// test removal
			//pub.RemoveSubscriber("sub1")
			//
			//// send another event, expect no delivery
			//ec <- deliver
			//select {
			//case <-received:
			//	t.Error("Subscriber received event after removal")
			//case <-time.After(2 * time.Second):
			//	// expected
			//}
			//time.Sleep(3 * time.Second) // allow time for processing
			//pub.DrainThenStop()
			synctest.Wait()

		})
	})
}

type SubscriberMock struct {
	id string
	c  chan string
	t  time.Duration
	l  *slog.Logger
	f  func(string)
}

func NewSubscriberMock(id string, t time.Duration) SubscriberMock {
	return SubscriberMock{
		id: id,
		c:  make(chan string),
		t:  t,
		l:  slog.Default(),
	}
}

func (s SubscriberMock) ID() string {
	return s.id
}

func (s SubscriberMock) Channel() chan<- string {
	return s.c
}

func (s SubscriberMock) TimeoutThreshold() time.Duration {
	return s.t
}

func (s SubscriberMock) SetHandler(f func(string)) {
	s.f = f
	go func() {
		for event := range s.c {
			s.l.Info(event)
			s.f(event)
		}
	}()
}
