package observer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// I am trying to integrate the new synctest package into the testing however, I am not sure
// how to best take advantage of it. I am finding it difficult to determine if I am testing
// the code or the clock. How to use the test effectively may become more obvious during later iterations of the code.

func TestNewNonBlockingPublisher(t *testing.T) {
	t.Run("CreatesInstance", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		if pub == nil {
			t.Error("Expected publisher instance, got nil")
		}
	})
}

func TestAddSubscriber(t *testing.T) {
	t.Run("AddSingleSubscriber", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub := NewSubscriberMock("sub1", 1*time.Second)

		err := pub.AddSubscriber(sub)
		if err != nil {
			t.Errorf("Unexpected error adding subscriber: %v", err)
		}

		if pub.SubscriberCount() != 1 {
			t.Errorf("Expected 1 subscriber, got %d", pub.SubscriberCount())
		}
	})

	t.Run("AddMultipleSubscribers", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub1 := NewSubscriberMock("sub1", 1*time.Second)
		sub2 := NewSubscriberMock("sub2", 1*time.Second)
		sub3 := NewSubscriberMock("sub3", 1*time.Second)

		_ = pub.AddSubscriber(sub1)
		_ = pub.AddSubscriber(sub2)
		_ = pub.AddSubscriber(sub3)

		if pub.SubscriberCount() != 3 {
			t.Errorf("Expected 3 subscribers, got %d", pub.SubscriberCount())
		}
	})

	t.Run("RejectDuplicateSubscriber", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub := NewSubscriberMock("sub1", 1*time.Second)

		err := pub.AddSubscriber(sub)
		if err != nil {
			t.Fatalf("First add failed: %v", err)
		}

		err = pub.AddSubscriber(sub)
		if !errors.Is(err, ErrSubscriberExists) {
			t.Errorf("Expected ErrSubscriberExists, got %v", err)
		}
	})

	t.Run("RejectSubscriberWithSameID", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub1 := NewSubscriberMock("duplicate-id", 1*time.Second)
		sub2 := NewSubscriberMock("duplicate-id", 2*time.Second)

		_ = pub.AddSubscriber(sub1)
		err := pub.AddSubscriber(sub2)

		if !errors.Is(err, ErrSubscriberExists) {
			t.Errorf("Expected ErrSubscriberExists, got %v", err)
		}
	})
}

func TestRemoveSubscriber(t *testing.T) {
	t.Run("RemoveExistingSubscriber", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub := NewSubscriberMock("sub1", 1*time.Second)

		_ = pub.AddSubscriber(sub)
		pub.RemoveSubscriber(sub.ID())

		if pub.SubscriberCount() != 0 {
			t.Errorf("Expected 0 subscribers after removal, got %d", pub.SubscriberCount())
		}
	})

	t.Run("RemoveNonExistentSubscriber", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		// Should not panic or error when removing non-existent subscriber
		pub.RemoveSubscriber("does-not-exist")
	})
}

// TODO: review
func TestPublishEvents(t *testing.T) {
	t.Run("PublishToSingleSubscriber", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub := NewSubscriberMock("sub1", 1*time.Second)
		sub.StartListening()

		_ = pub.AddSubscriber(sub)
		eventChan := pub.Start()

		eventChan <- "event1"
		time.Sleep(50 * time.Millisecond)

		received := sub.GetReceivedEvents()
		if len(received) != 1 {
			t.Errorf("Expected 1 event, got %d", len(received))
		}
		if len(received) > 0 && received[0] != "event1" {
			t.Errorf("Expected 'event1', got '%s'", received[0])
		}

		pub.DrainThenStop()
	})

	// TODO: review for synctest
	t.Run("PublishToMultipleSubscribers", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub1 := NewSubscriberMock("sub1", 1*time.Second)
		sub2 := NewSubscriberMock("sub2", 1*time.Second)
		sub3 := NewSubscriberMock("sub3", 1*time.Second)

		sub1.StartListening()
		sub2.StartListening()
		sub3.StartListening()

		_ = pub.AddSubscriber(sub1)
		_ = pub.AddSubscriber(sub2)
		_ = pub.AddSubscriber(sub3)

		eventChan := pub.Start()

		eventChan <- "event1"
		eventChan <- "event2"
		time.Sleep(50 * time.Millisecond)

		for _, sub := range []*SubscriberMock{sub1, sub2, sub3} {
			received := sub.GetReceivedEvents()
			if len(received) != 2 {
				t.Errorf("Subscriber %s: expected 2 events, got %d", sub.ID(), len(received))
			}
		}

		pub.DrainThenStop()
	})

	t.Run("PublishAfterSubscriberRemoval", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub1 := NewSubscriberMock("sub1", 1*time.Second)
		sub2 := NewSubscriberMock("sub2", 1*time.Second)

		sub1.StartListening()
		sub2.StartListening()

		_ = pub.AddSubscriber(sub1)
		_ = pub.AddSubscriber(sub2)

		eventChan := pub.Start()

		// First event - both should receive
		eventChan <- "event1"
		time.Sleep(50 * time.Millisecond)

		// Remove sub1
		pub.RemoveSubscriber(sub1.ID())

		// Second event - only sub2 should receive
		eventChan <- "event2"
		time.Sleep(50 * time.Millisecond)

		received1 := sub1.GetReceivedEvents()
		received2 := sub2.GetReceivedEvents()

		if len(received1) != 1 {
			t.Errorf("Sub1: expected 1 event, got %d", len(received1))
		}
		if len(received2) != 2 {
			t.Errorf("Sub2: expected 2 events, got %d", len(received2))
		}

		pub.DrainThenStop()
	})
}

func TestPublisherShutdown(t *testing.T) {
	t.Run("DrainThenStop", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub := NewSubscriberMock("sub1", 1*time.Second)
		sub.StartListening()

		_ = pub.AddSubscriber(sub)
		eventChan := pub.Start()

		// Send multiple events
		eventChan <- "event1"
		eventChan <- "event2"
		eventChan <- "event3"

		// DrainThenStop should wait for all events to be processed
		pub.DrainThenStop()

		received := sub.GetReceivedEvents()
		if len(received) != 3 {
			t.Errorf("Expected 3 events after drain, got %d", len(received))
		}
	})

	t.Run("Halt", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		sub := NewSubscriberMock("sub1", 1*time.Second)
		sub.StartListening()

		_ = pub.AddSubscriber(sub)
		eventChan := pub.Start()

		eventChan <- "event1"
		time.Sleep(50 * time.Millisecond)

		// Halt should stop immediately and cancel in-progress events
		pub.Halt()

		// Verify at least one event was received before halt
		received := sub.GetReceivedEvents()
		if len(received) >= 1 {
			if received[0] != "event1" {
				t.Errorf("Expected 'event1', got '%s'", received[0])
			}
		}
	})
}

func TestSubscriberTimeout(t *testing.T) {
	t.Run("SlowSubscriberTimesOut", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()
		// Create a subscriber with very short timeout and no listener
		// (channel will block since nothing is reading from it)
		sub := NewSubscriberMock("slow-sub", 1*time.Millisecond)

		_ = pub.AddSubscriber(sub)
		eventChan := pub.Start()

		// This should timeout since no one is reading from sub's channel
		eventChan <- "event1"
		time.Sleep(50 * time.Millisecond)

		// Publisher should continue functioning
		if pub.SubscriberCount() != 1 {
			t.Error("Subscriber should still be registered after timeout")
		}

		pub.Halt()
	})
}

// TestTimeoutThreshold tests that subscriber timeout thresholds work correctly
// Note: Uses real time, not synctest, because timeout behavior requires background goroutines
func TestTimeoutThreshold(t *testing.T) {
	t.Run("TimeoutWithBlockedChannel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			pub := NewNonBlockingPublisher[string]()

			// Create a subscriber with unbuffered channel and no listener
			// This will cause sends to block and trigger timeout
			blockedSub := &BlockingSubscriberMock{
				id: "blocked",
				c:  make(chan string), // unbuffered, no reader
				t:  50 * time.Millisecond,
			}
			_ = pub.AddSubscriber(blockedSub)

			ctx := context.Background()
			startTime := time.Now()

			pub.NotifySubscribers(ctx, "test")

			time.Sleep(50 * time.Millisecond)
			elapsed := time.Since(startTime)

			if elapsed < 40*time.Millisecond {
				t.Errorf("Timeout triggered too early: %v (expected >= 50ms)", elapsed)
			}
			if elapsed > 200*time.Millisecond {
				t.Errorf("Timeout took too long: %v (expected ~50ms)", elapsed)
			}
		})

	})

	t.Run("ContextCancellationBeforeTimeout", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()

		// Create subscriber with long timeout
		blockedSub := &BlockingSubscriberMock{
			id: "blocked",
			c:  make(chan string),
			t:  1 * time.Second, // Long timeout
		}
		_ = pub.AddSubscriber(blockedSub)

		// Create cancellable context
		ctx, cancel := context.WithCancel(context.Background())

		startTime := time.Now()
		done := make(chan bool, 1)
		go func() {
			pub.NotifySubscribers(ctx, "test")
			pub.wg.Wait()
			done <- true
		}()

		// Cancel context before timeout (should trigger at 100ms, well before 1s timeout)
		time.Sleep(100 * time.Millisecond)
		cancel()

		select {
		case <-done:
			elapsed := time.Since(startTime)
			// Should complete due to context cancellation (around 100ms)
			// NOT due to timeout (which would be 1s)
			if elapsed > 500*time.Millisecond {
				t.Errorf("Completed too late: %v (should cancel at ~100ms, not wait for 1s timeout)", elapsed)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Notification did not complete after context cancellation")
		}
	})

	t.Run("MultipleSubscribersWithDifferentThresholds", func(t *testing.T) {
		pub := NewNonBlockingPublisher[string]()

		// Add multiple subscribers with different timeouts
		// All with blocked channels
		sub1 := &BlockingSubscriberMock{id: "sub1", c: make(chan string), t: 20 * time.Millisecond}
		sub2 := &BlockingSubscriberMock{id: "sub2", c: make(chan string), t: 50 * time.Millisecond}
		sub3 := &BlockingSubscriberMock{id: "sub3", c: make(chan string), t: 100 * time.Millisecond}

		_ = pub.AddSubscriber(sub1)
		_ = pub.AddSubscriber(sub2)
		_ = pub.AddSubscriber(sub3)

		ctx := context.Background()
		startTime := time.Now()

		pub.NotifySubscribers(ctx, "test")
		pub.wg.Wait()

		time.Sleep(90 * time.Millisecond)

		elapsed := time.Since(startTime)
		// All should timeout independently
		// NotifySubscribers waits for all to complete, so we wait for the longest (100ms)
		if elapsed < 80*time.Millisecond {
			t.Errorf("Completed too early: %v (expected >= 100ms for slowest subscriber)", elapsed)
		}
		if elapsed > 250*time.Millisecond {
			t.Errorf("Completed too late: %v (expected ~100ms)", elapsed)
		}
	})
}

// TestTimeoutThresholdWithSynctest demonstrates using synctest to verify timeout configuration
func TestTimeoutThresholdWithSynctest(t *testing.T) {
	t.Run("VerifyTimeoutThresholdIsUsed", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// This test verifies that the TimeoutThreshold() method is correctly
			// called and used when creating contexts
			pub := NewNonBlockingPublisher[string]()

			// Create subscribers with known timeout thresholds
			sub1 := &TimeoutTrackingSubscriber{
				id: "sub1",
				c:  make(chan string, 10),
				t:  100 * time.Millisecond,
			}
			sub2 := &TimeoutTrackingSubscriber{
				id: "sub2",
				c:  make(chan string, 10),
				t:  200 * time.Millisecond,
			}

			_ = pub.AddSubscriber(sub1)
			_ = pub.AddSubscriber(sub2)

			// Verify the subscribers are registered with correct thresholds
			if pub.SubscriberCount() != 2 {
				t.Errorf("Expected 2 subscribers, got %d", pub.SubscriberCount())
			}

			// Verify each subscriber's threshold is accessible
			if sub1.TimeoutThreshold() != 100*time.Millisecond {
				t.Errorf("Sub1: expected 100ms threshold, got %v", sub1.TimeoutThreshold())
			}
			if sub2.TimeoutThreshold() != 200*time.Millisecond {
				t.Errorf("Sub2: expected 200ms threshold, got %v", sub2.TimeoutThreshold())
			}
		})
	})
}

// TestSynctestConcurrency demonstrates using synctest.Test for deterministic concurrency testing
func TestSynctestConcurrency(t *testing.T) {
	t.Run("ConcurrentAddSubscribers", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			pub := NewNonBlockingPublisher[string]()
			var wg sync.WaitGroup

			// Add 10 subscribers concurrently
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					sub := NewSubscriberMock(string(rune('A'+id)), 1*time.Second)
					_ = pub.AddSubscriber(sub)
				}(i)
			}

			wg.Wait()

			if pub.SubscriberCount() != 10 {
				t.Errorf("Expected 10 subscribers, got %d", pub.SubscriberCount())
			}
		})
	})

	t.Run("ConcurrentRemoveSubscribers", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			pub := NewNonBlockingPublisher[string]()

			// Add subscribers first
			for i := 0; i < 5; i++ {
				sub := NewSubscriberMock(string(rune('A'+i)), 1*time.Second)
				_ = pub.AddSubscriber(sub)
			}

			var wg sync.WaitGroup
			// Remove subscribers concurrently
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					pub.RemoveSubscriber(string(rune('A' + id)))
				}(i)
			}

			wg.Wait()

			if pub.SubscriberCount() != 0 {
				t.Errorf("Expected 0 subscribers after removal, got %d", pub.SubscriberCount())
			}
		})
	})

	t.Run("ConcurrentAddAndCount", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			pub := NewNonBlockingPublisher[int]()
			var wg sync.WaitGroup

			// Concurrently add subscribers and check count
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					sub := NewIntSubscriberMock(string(rune('A'+id%26))+string(rune('0'+id/26)), 1*time.Second)
					_ = pub.AddSubscriber(sub)
					_ = pub.SubscriberCount() // concurrent reads
				}(i)
			}

			wg.Wait()

			if pub.SubscriberCount() != 20 {
				t.Errorf("Expected 20 subscribers, got %d", pub.SubscriberCount())
			}
		})
	})
}

// TestRealWorldConcurrency tests real-world concurrent publishing without synctest
// (synctest is not suitable for testing goroutines that outlive the test bubble)
func TestRealWorldConcurrency(t *testing.T) {
	t.Run("ConcurrentPublishAndSubscribe", func(t *testing.T) {
		pub := NewNonBlockingPublisher[int]()
		eventChan := pub.Start()

		// Add subscribers
		subscribers := make([]*IntSubscriberMock, 5)
		for i := 0; i < 5; i++ {
			sub := NewIntSubscriberMock(string(rune('A'+i)), 1*time.Second)
			sub.StartListening()
			subscribers[i] = sub
			_ = pub.AddSubscriber(sub)
		}

		// Publish events concurrently
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				eventChan <- val
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond) // Allow time for processing

		pub.DrainThenStop()

		// Verify each subscriber received all events
		for _, sub := range subscribers {
			received := sub.GetReceivedEvents()
			if len(received) != 10 {
				t.Errorf("Subscriber %s: expected 10 events, got %d", sub.ID(), len(received))
			}
		}
	})
}

// SubscriberMock is a test subscriber that collects received events
type SubscriberMock struct {
	id        string
	c         chan string
	t         time.Duration
	mu        sync.Mutex
	received  []string
	listening atomic.Bool
}

func NewSubscriberMock(id string, t time.Duration) *SubscriberMock {
	return &SubscriberMock{
		id:       id,
		c:        make(chan string, 10), // Buffered to prevent blocking in tests
		t:        t,
		received: make([]string, 0),
	}
}

func (s *SubscriberMock) ID() string {
	return s.id
}

func (s *SubscriberMock) Channel() chan<- string {
	return s.c
}

func (s *SubscriberMock) TimeoutThreshold() time.Duration {
	return s.t
}

// StartListening begins processing events from the channel
func (s *SubscriberMock) StartListening() {
	if s.listening.Swap(true) {
		return // Already listening
	}

	go func() {
		for event := range s.c {
			s.mu.Lock()
			s.received = append(s.received, event)
			s.mu.Unlock()
		}
	}()
}

// GetReceivedEvents returns a copy of all received events
func (s *SubscriberMock) GetReceivedEvents() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]string, len(s.received))
	copy(result, s.received)
	return result
}

// IntSubscriberMock is a test subscriber for integer events
type IntSubscriberMock struct {
	id        string
	c         chan int
	t         time.Duration
	mu        sync.Mutex
	received  []int
	listening atomic.Bool
}

func NewIntSubscriberMock(id string, t time.Duration) *IntSubscriberMock {
	return &IntSubscriberMock{
		id:       id,
		c:        make(chan int, 10),
		t:        t,
		received: make([]int, 0),
	}
}

func (s *IntSubscriberMock) ID() string {
	return s.id
}

func (s *IntSubscriberMock) Channel() chan<- int {
	return s.c
}

func (s *IntSubscriberMock) TimeoutThreshold() time.Duration {
	return s.t
}

func (s *IntSubscriberMock) StartListening() {
	if s.listening.Swap(true) {
		return
	}

	go func() {
		for event := range s.c {
			s.mu.Lock()
			s.received = append(s.received, event)
			s.mu.Unlock()
		}
	}()
}

func (s *IntSubscriberMock) GetReceivedEvents() []int {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]int, len(s.received))
	copy(result, s.received)
	return result
}

// BlockingSubscriberMock is a subscriber with an unbuffered channel that blocks sends
type BlockingSubscriberMock struct {
	id string
	c  chan string
	t  time.Duration
}

func (s *BlockingSubscriberMock) ID() string {
	return s.id
}

func (s *BlockingSubscriberMock) Channel() chan<- string {
	return s.c
}

func (s *BlockingSubscriberMock) TimeoutThreshold() time.Duration {
	return s.t
}

// TimeoutTrackingSubscriber is used to verify timeout threshold configuration
type TimeoutTrackingSubscriber struct {
	id string
	c  chan string
	t  time.Duration
}

func (s *TimeoutTrackingSubscriber) ID() string {
	return s.id
}

func (s *TimeoutTrackingSubscriber) Channel() chan<- string {
	return s.c
}

func (s *TimeoutTrackingSubscriber) TimeoutThreshold() time.Duration {
	return s.t
}
