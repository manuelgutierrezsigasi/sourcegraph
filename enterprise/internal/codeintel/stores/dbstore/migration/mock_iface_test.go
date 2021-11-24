// Code generated by go-mockgen 1.1.2; DO NOT EDIT.

package migration

import (
	"context"
	"sync"
	"time"
)

// MockGitserverClient is a mock implementation of the GitserverClient
// interface (from the package
// github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore/migration)
// used for unit testing.
type MockGitserverClient struct {
	// CommitDateFunc is an instance of a mock function object controlling
	// the behavior of the method CommitDate.
	CommitDateFunc *GitserverClientCommitDateFunc
}

// NewMockGitserverClient creates a new mock of the GitserverClient
// interface. All methods return zero values for all results, unless
// overwritten.
func NewMockGitserverClient() *MockGitserverClient {
	return &MockGitserverClient{
		CommitDateFunc: &GitserverClientCommitDateFunc{
			defaultHook: func(context.Context, int, string) (string, time.Time, bool, error) {
				return "", time.Time{}, false, nil
			},
		},
	}
}

// NewStrictMockGitserverClient creates a new mock of the GitserverClient
// interface. All methods panic on invocation, unless overwritten.
func NewStrictMockGitserverClient() *MockGitserverClient {
	return &MockGitserverClient{
		CommitDateFunc: &GitserverClientCommitDateFunc{
			defaultHook: func(context.Context, int, string) (string, time.Time, bool, error) {
				panic("unexpected invocation of MockGitserverClient.CommitDate")
			},
		},
	}
}

// NewMockGitserverClientFrom creates a new mock of the MockGitserverClient
// interface. All methods delegate to the given implementation, unless
// overwritten.
func NewMockGitserverClientFrom(i GitserverClient) *MockGitserverClient {
	return &MockGitserverClient{
		CommitDateFunc: &GitserverClientCommitDateFunc{
			defaultHook: i.CommitDate,
		},
	}
}

// GitserverClientCommitDateFunc describes the behavior when the CommitDate
// method of the parent MockGitserverClient instance is invoked.
type GitserverClientCommitDateFunc struct {
	defaultHook func(context.Context, int, string) (string, time.Time, bool, error)
	hooks       []func(context.Context, int, string) (string, time.Time, bool, error)
	history     []GitserverClientCommitDateFuncCall
	mutex       sync.Mutex
}

// CommitDate delegates to the next hook function in the queue and stores
// the parameter and result values of this invocation.
func (m *MockGitserverClient) CommitDate(v0 context.Context, v1 int, v2 string) (string, time.Time, bool, error) {
	r0, r1, r2, r3 := m.CommitDateFunc.nextHook()(v0, v1, v2)
	m.CommitDateFunc.appendCall(GitserverClientCommitDateFuncCall{v0, v1, v2, r0, r1, r2, r3})
	return r0, r1, r2, r3
}

// SetDefaultHook sets function that is called when the CommitDate method of
// the parent MockGitserverClient instance is invoked and the hook queue is
// empty.
func (f *GitserverClientCommitDateFunc) SetDefaultHook(hook func(context.Context, int, string) (string, time.Time, bool, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// CommitDate method of the parent MockGitserverClient instance invokes the
// hook at the front of the queue and discards it. After the queue is empty,
// the default hook function is invoked for any future action.
func (f *GitserverClientCommitDateFunc) PushHook(hook func(context.Context, int, string) (string, time.Time, bool, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *GitserverClientCommitDateFunc) SetDefaultReturn(r0 string, r1 time.Time, r2 bool, r3 error) {
	f.SetDefaultHook(func(context.Context, int, string) (string, time.Time, bool, error) {
		return r0, r1, r2, r3
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *GitserverClientCommitDateFunc) PushReturn(r0 string, r1 time.Time, r2 bool, r3 error) {
	f.PushHook(func(context.Context, int, string) (string, time.Time, bool, error) {
		return r0, r1, r2, r3
	})
}

func (f *GitserverClientCommitDateFunc) nextHook() func(context.Context, int, string) (string, time.Time, bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *GitserverClientCommitDateFunc) appendCall(r0 GitserverClientCommitDateFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of GitserverClientCommitDateFuncCall objects
// describing the invocations of this function.
func (f *GitserverClientCommitDateFunc) History() []GitserverClientCommitDateFuncCall {
	f.mutex.Lock()
	history := make([]GitserverClientCommitDateFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// GitserverClientCommitDateFuncCall is an object that describes an
// invocation of method CommitDate on an instance of MockGitserverClient.
type GitserverClientCommitDateFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 string
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 string
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 time.Time
	// Result2 is the value of the 3rd result returned from this method
	// invocation.
	Result2 bool
	// Result3 is the value of the 4th result returned from this method
	// invocation.
	Result3 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c GitserverClientCommitDateFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c GitserverClientCommitDateFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1, c.Result2, c.Result3}
}
