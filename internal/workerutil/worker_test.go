package workerutil

import (
	"context"
	"fmt"
	"sync"

	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/derision-test/glock"

	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type TestRecord struct {
	ID    int
	State string
}

func (v TestRecord) RecordID() int {
	return v.ID
}

func TestWorkerHandlerSuccess(t *testing.T) {
	store := NewMockStore()
	handler := NewMockHandler()
	dequeueClock := glock.NewMockClock()
	heartbeatClock := glock.NewMockClock()
	shutdownClock := glock.NewMockClock()
	options := WorkerOptions{
		Name:           "test",
		WorkerHostname: "test",
		NumHandlers:    1,
		Interval:       time.Second,
		Metrics:        NewMetrics(&observation.TestContext, "", nil),
	}

	store.DequeueFunc.PushReturn(TestRecord{ID: 42}, true, nil)
	store.DequeueFunc.SetDefaultReturn(nil, false, nil)
	store.MarkCompleteFunc.SetDefaultReturn(true, nil)

	worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
	go func() { worker.Start() }()
	dequeueClock.BlockingAdvance(time.Second)
	worker.Stop()

	if callCount := len(handler.HandleFunc.History()); callCount != 1 {
		t.Errorf("unexpected handle call count. want=%d have=%d", 1, callCount)
	} else if arg := handler.HandleFunc.History()[0].Arg1; arg.RecordID() != 42 {
		t.Errorf("unexpected record. want=%d have=%d", 42, arg.RecordID())
	}

	if callCount := len(store.MarkCompleteFunc.History()); callCount != 1 {
		t.Errorf("unexpected mark complete call count. want=%d have=%d", 1, callCount)
	} else if id := store.MarkCompleteFunc.History()[0].Arg1; id != 42 {
		t.Errorf("unexpected id argument to mark complete. want=%v have=%v", 42, id)
	}
}

func TestWorkerHandlerFailure(t *testing.T) {
	store := NewMockStore()
	handler := NewMockHandler()
	dequeueClock := glock.NewMockClock()
	heartbeatClock := glock.NewMockClock()
	shutdownClock := glock.NewMockClock()
	options := WorkerOptions{
		Name:           "test",
		WorkerHostname: "test",
		NumHandlers:    1,
		Interval:       time.Second,
		Metrics:        NewMetrics(&observation.TestContext, "", nil),
	}

	store.DequeueFunc.PushReturn(TestRecord{ID: 42}, true, nil)
	store.DequeueFunc.SetDefaultReturn(nil, false, nil)
	store.MarkErroredFunc.SetDefaultReturn(true, nil)
	handler.HandleFunc.SetDefaultReturn(errors.Errorf("oops"))

	worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
	go func() { worker.Start() }()
	dequeueClock.BlockingAdvance(time.Second)
	worker.Stop()

	if callCount := len(handler.HandleFunc.History()); callCount != 1 {
		t.Errorf("unexpected handle call count. want=%d have=%d", 1, callCount)
	} else if arg := handler.HandleFunc.History()[0].Arg1; arg.RecordID() != 42 {
		t.Errorf("unexpected record. want=%d have=%d", 42, arg.RecordID())
	}

	if callCount := len(store.MarkErroredFunc.History()); callCount != 1 {
		t.Errorf("unexpected mark errored call count. want=%d have=%d", 1, callCount)
	} else if id := store.MarkErroredFunc.History()[0].Arg1; id != 42 {
		t.Errorf("unexpected id argument to mark errored. want=%v have=%v", 42, id)
	} else if failureMessage := store.MarkErroredFunc.History()[0].Arg2; failureMessage != "oops" {
		t.Errorf("unexpected failure message argument to mark errored. want=%q have=%q", "oops", failureMessage)
	}
}

type nonRetryableTestErr struct{}

func (e nonRetryableTestErr) Error() string      { return "just retry me and see what happens" }
func (e nonRetryableTestErr) NonRetryable() bool { return true }

func TestWorkerHandlerNonRetryableFailure(t *testing.T) {
	store := NewMockStore()
	handler := NewMockHandler()
	dequeueClock := glock.NewMockClock()
	heartbeatClock := glock.NewMockClock()
	shutdownClock := glock.NewMockClock()
	options := WorkerOptions{
		Name:           "test",
		WorkerHostname: "test",
		NumHandlers:    1,
		Interval:       time.Second,
		Metrics:        NewMetrics(&observation.TestContext, "", nil),
	}

	store.DequeueFunc.PushReturn(TestRecord{ID: 42}, true, nil)
	store.DequeueFunc.SetDefaultReturn(nil, false, nil)
	store.MarkFailedFunc.SetDefaultReturn(true, nil)

	testErr := nonRetryableTestErr{}
	handler.HandleFunc.SetDefaultReturn(testErr)

	worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
	go func() { worker.Start() }()
	dequeueClock.BlockingAdvance(time.Second)
	worker.Stop()

	if callCount := len(handler.HandleFunc.History()); callCount != 1 {
		t.Errorf("unexpected handle call count. want=%d have=%d", 1, callCount)
	} else if arg := handler.HandleFunc.History()[0].Arg1; arg.RecordID() != 42 {
		t.Errorf("unexpected record. want=%d have=%d", 42, arg.RecordID())
	}

	if callCount := len(store.MarkFailedFunc.History()); callCount != 1 {
		t.Errorf("unexpected mark failed call count. want=%d have=%d", 1, callCount)
	} else if id := store.MarkFailedFunc.History()[0].Arg1; id != 42 {
		t.Errorf("unexpected id argument to mark failed. want=%v have=%v", 42, id)
	} else if failureMessage := store.MarkFailedFunc.History()[0].Arg2; failureMessage != testErr.Error() {
		t.Errorf("unexpected failure message argument to mark failed. want=%q have=%q", testErr.Error(), failureMessage)
	}
}

func TestWorkerConcurrent(t *testing.T) {
	t.Skip("Disabled because it's flaky. See: https://github.com/sourcegraph/sourcegraph/issues/22595")

	NumTestRecords := 50

	for numHandlers := 1; numHandlers < NumTestRecords; numHandlers++ {
		name := fmt.Sprintf("numHandlers=%d", numHandlers)

		t.Run(name, func(t *testing.T) {
			store := NewMockStore()
			handler := NewMockHandlerWithHooks()
			dequeueClock := glock.NewMockClock()
			heartbeatClock := glock.NewMockClock()
			shutdownClock := glock.NewMockClock()
			options := WorkerOptions{
				Name:           "test",
				WorkerHostname: "test",
				NumHandlers:    numHandlers,
				Interval:       time.Second,
				Metrics:        NewMetrics(&observation.TestContext, "", nil),
			}

			for i := 0; i < NumTestRecords; i++ {
				index := i
				store.DequeueFunc.PushReturn(TestRecord{ID: index}, true, nil)
			}
			store.DequeueFunc.SetDefaultReturn(nil, false, nil)

			var m sync.Mutex
			times := map[int][2]time.Time{}
			markTime := func(recordID, index int) {
				m.Lock()
				pair := times[recordID]
				pair[index] = time.Now()
				times[recordID] = pair
				m.Unlock()
			}

			handler.PreHandleFunc.SetDefaultHook(func(ctx context.Context, record Record) { markTime(record.RecordID(), 0) })
			handler.PostHandleFunc.SetDefaultHook(func(ctx context.Context, record Record) { markTime(record.RecordID(), 1) })
			handler.HandleFunc.SetDefaultHook(func(context.Context, Record) error {
				// Do a _very_ small sleep to make it very unlikely that the scheduler
				// will happen to invoke all of the handlers sequentially.
				<-time.After(time.Millisecond * 10)
				return nil
			})

			worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
			go func() { worker.Start() }()
			for i := 0; i < NumTestRecords; i++ {
				dequeueClock.BlockingAdvance(time.Second)
			}
			worker.Stop()

			intersecting := 0
			for i := 0; i < NumTestRecords; i++ {
				for j := i + 1; j < NumTestRecords; j++ {
					if !times[1][1].Before(times[j][0]) {
						if j-i > 2*numHandlers-1 {
							// The greatest distance between two "batches" that can overlap is
							// just under 2x the number of concurrent handler routines. For example
							// if n=3:
							//
							// t1: dequeue A (1 active) *
							// t2: dequeue B (2 active)
							// t3: dequeue C (3 active)
							// t4: process C (2 active)
							// t5: dequeue D (3 active)
							// t6: process B (2 active)
							// t7: dequeue E (3 active) *
							// t8: process A (2 active) *
							//
							// Here, A finishes after E is dequeued, which has a distance of 5 (2*3-1).

							t.Errorf(
								"times %[1]d (%[3]s-%[4]s) and %[2]d (%[5]s-%[6]s) failed validation",
								i,
								j,
								times[i][0],
								times[i][1],
								times[j][0],
								times[j][1],
							)
						}

						intersecting++
					}
				}
			}

			if numHandlers > 1 && intersecting == 0 {
				t.Errorf("no handler routines were concurrent")
			}
		})
	}
}

func TestWorkerBlockingPreDequeueHook(t *testing.T) {
	store := NewMockStore()
	handler := NewMockHandlerWithPreDequeue()
	dequeueClock := glock.NewMockClock()
	heartbeatClock := glock.NewMockClock()
	shutdownClock := glock.NewMockClock()
	options := WorkerOptions{
		Name:           "test",
		WorkerHostname: "test",
		NumHandlers:    1,
		Interval:       time.Second,
		Metrics:        NewMetrics(&observation.TestContext, "", nil),
	}

	store.DequeueFunc.PushReturn(TestRecord{ID: 42}, true, nil)
	store.DequeueFunc.SetDefaultReturn(nil, false, nil)

	// Block all dequeues
	handler.PreDequeueFunc.SetDefaultReturn(false, nil, nil)

	worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
	go func() { worker.Start() }()
	dequeueClock.BlockingAdvance(time.Second)
	worker.Stop()

	if callCount := len(handler.HandleFunc.History()); callCount != 0 {
		t.Errorf("unexpected handle call count. want=%d have=%d", 0, callCount)
	}
}

func TestWorkerConditionalPreDequeueHook(t *testing.T) {
	store := NewMockStore()
	handler := NewMockHandlerWithPreDequeue()
	dequeueClock := glock.NewMockClock()
	heartbeatClock := glock.NewMockClock()
	shutdownClock := glock.NewMockClock()
	options := WorkerOptions{
		Name:           "test",
		WorkerHostname: "test",
		NumHandlers:    1,
		Interval:       time.Second,
		Metrics:        NewMetrics(&observation.TestContext, "", nil),
	}

	store.DequeueFunc.PushReturn(TestRecord{ID: 42}, true, nil)
	store.DequeueFunc.PushReturn(TestRecord{ID: 43}, true, nil)
	store.DequeueFunc.PushReturn(TestRecord{ID: 44}, true, nil)
	store.DequeueFunc.SetDefaultReturn(nil, false, nil)

	// Return additional arguments
	handler.PreDequeueFunc.PushReturn(true, "A", nil)
	handler.PreDequeueFunc.PushReturn(true, "B", nil)
	handler.PreDequeueFunc.PushReturn(true, "C", nil)

	worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
	go func() { worker.Start() }()
	dequeueClock.BlockingAdvance(time.Second)
	dequeueClock.BlockingAdvance(time.Second)
	dequeueClock.BlockingAdvance(time.Second)
	worker.Stop()

	if callCount := len(handler.HandleFunc.History()); callCount != 3 {
		t.Errorf("unexpected handle call count. want=%d have=%d", 3, callCount)
	}

	if callCount := len(store.DequeueFunc.History()); callCount != 3 {
		t.Errorf("unexpected dequeue call count. want=%d have=%d", 3, callCount)
	} else {
		for i, expected := range []string{"A", "B", "C"} {
			if extra := store.DequeueFunc.History()[i].Arg2; extra != expected {
				t.Errorf("unexpected extra argument for dequeue call %d. want=%q have=%q", i, expected, extra)
			}
		}
	}
}

type MockHandlerWithPreDequeue struct {
	*MockHandler
	*MockWithPreDequeue
}

func NewMockHandlerWithPreDequeue() *MockHandlerWithPreDequeue {
	return &MockHandlerWithPreDequeue{
		MockHandler:        NewMockHandler(),
		MockWithPreDequeue: NewMockWithPreDequeue(),
	}
}

type MockHandlerWithHooks struct {
	*MockHandler
	*MockWithHooks
}

func NewMockHandlerWithHooks() *MockHandlerWithHooks {
	return &MockHandlerWithHooks{
		MockHandler:   NewMockHandler(),
		MockWithHooks: NewMockWithHooks(),
	}
}

func TestWorkerDequeueHeartbeat(t *testing.T) {
	store := NewMockStore()
	store.DequeueFunc.PushReturn(TestRecord{ID: 42}, true, nil)
	store.DequeueFunc.SetDefaultReturn(nil, false, nil)
	store.MarkCompleteFunc.SetDefaultReturn(true, nil)

	handler := NewMockHandler()
	dequeueClock := glock.NewMockClock()
	heartbeatClock := glock.NewMockClock()
	shutdownClock := glock.NewMockClock()
	heartbeatInterval := time.Second
	options := WorkerOptions{
		Name:              "test",
		WorkerHostname:    "test",
		NumHandlers:       1,
		HeartbeatInterval: heartbeatInterval,
		Interval:          time.Second,
		Metrics:           NewMetrics(&observation.TestContext, "", nil),
	}

	dequeued := make(chan struct{})
	doneHandling := make(chan struct{})
	handler.HandleFunc.defaultHook = func(c context.Context, r Record) error {
		close(dequeued)
		<-doneHandling
		return nil
	}

	heartbeats := make(chan struct{})
	store.HeartbeatFunc.SetDefaultHook(func(c context.Context, i []int) ([]int, error) {
		heartbeats <- struct{}{}
		return i, nil
	})

	worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
	go func() { worker.Start() }()
	t.Cleanup(func() {
		close(doneHandling)
		worker.Stop()
	})
	<-dequeued

	for range []int{1, 2, 3, 4, 5} {
		heartbeatClock.BlockingAdvance(heartbeatInterval)
		select {
		case <-heartbeats:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for heartbeat")
		}
	}
}

func TestWorkerNumTotalJobs(t *testing.T) {
	store := NewMockStore()
	handler := NewMockHandler()
	dequeueClock := glock.NewMockClock()
	heartbeatClock := glock.NewMockClock()
	shutdownClock := glock.NewMockClock()
	options := WorkerOptions{
		Name:           "test",
		WorkerHostname: "test",
		NumHandlers:    1,
		NumTotalJobs:   5,
		Interval:       time.Second,
		Metrics:        NewMetrics(&observation.TestContext, "", nil),
	}

	store.DequeueFunc.SetDefaultReturn(TestRecord{ID: 42}, true, nil)
	store.MarkCompleteFunc.SetDefaultReturn(true, nil)

	// Should process 5 then shut down
	worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
	worker.Start()

	if callCount := len(store.DequeueFunc.History()); callCount != 5 {
		t.Errorf("unexpected call count. want=%d have=%d", 5, callCount)
	}
}

func TestWorkerMaxActiveTime(t *testing.T) {
	store := NewMockStore()
	handler := NewMockHandler()
	dequeueClock := glock.NewMockClock()
	heartbeatClock := glock.NewMockClock()
	shutdownClock := glock.NewMockClock()
	options := WorkerOptions{
		Name:           "test",
		WorkerHostname: "test",
		NumHandlers:    1,
		NumTotalJobs:   50,
		MaxActiveTime:  time.Second * 5,
		Interval:       time.Second,
		Metrics:        NewMetrics(&observation.TestContext, "", nil),
	}

	called := make(chan struct{})
	defer close(called)

	dequeueHook := func(c context.Context, s string, i interface{}) (Record, bool, error) {
		called <- struct{}{}
		return TestRecord{ID: 42}, true, nil
	}

	store.DequeueFunc.SetDefaultReturn(nil, false, nil)
	store.MarkCompleteFunc.SetDefaultReturn(true, nil)

	for i := 0; i < 5; i++ {
		store.DequeueFunc.PushHook(dequeueHook)
	}

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		worker := newWorker(context.Background(), store, handler, options, dequeueClock, heartbeatClock, shutdownClock)
		worker.Start()
	}()

	timeout := time.After(time.Second * 5)
	for i := 0; i < 5; i++ {
		select {
		case <-timeout:
			t.Fatal("timeout waiting for dequeues")
		case <-called:
		}
	}

	// Wait for a fixed number of records to be processed, then
	// send a shutdown signal based on time after that. If the
	// goroutine running the worker is released, then it's has
	// shut down properly.
	shutdownClock.BlockingAdvance(time.Second * 5)

	select {
	case <-timeout:
		t.Fatal("timeout waiting for shutdown")
	case <-stopped:
	}

	// Might dequeue 5 or 6 based on timing
	if callCount := len(store.DequeueFunc.History()); callCount > 5 {
		t.Errorf("unexpected call count. want<=%d have=%d", 5, callCount)
	}
}
