// Code generated by go-mockgen 1.1.2; DO NOT EDIT.

package authz

import (
	"context"
	api "github.com/sourcegraph/sourcegraph/internal/api"
	"sync"
)

// MockSubRepoPermissionsGetter is a mock implementation of the
// SubRepoPermissionsGetter interface (from the package
// github.com/sourcegraph/sourcegraph/internal/authz) used for unit testing.
type MockSubRepoPermissionsGetter struct {
	// GetByUserFunc is an instance of a mock function object controlling
	// the behavior of the method GetByUser.
	GetByUserFunc *SubRepoPermissionsGetterGetByUserFunc
}

// NewMockSubRepoPermissionsGetter creates a new mock of the
// SubRepoPermissionsGetter interface. All methods return zero values for
// all results, unless overwritten.
func NewMockSubRepoPermissionsGetter() *MockSubRepoPermissionsGetter {
	return &MockSubRepoPermissionsGetter{
		GetByUserFunc: &SubRepoPermissionsGetterGetByUserFunc{
			defaultHook: func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error) {
				return nil, nil
			},
		},
	}
}

// NewStrictMockSubRepoPermissionsGetter creates a new mock of the
// SubRepoPermissionsGetter interface. All methods panic on invocation,
// unless overwritten.
func NewStrictMockSubRepoPermissionsGetter() *MockSubRepoPermissionsGetter {
	return &MockSubRepoPermissionsGetter{
		GetByUserFunc: &SubRepoPermissionsGetterGetByUserFunc{
			defaultHook: func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error) {
				panic("unexpected invocation of MockSubRepoPermissionsGetter.GetByUser")
			},
		},
	}
}

// NewMockSubRepoPermissionsGetterFrom creates a new mock of the
// MockSubRepoPermissionsGetter interface. All methods delegate to the given
// implementation, unless overwritten.
func NewMockSubRepoPermissionsGetterFrom(i SubRepoPermissionsGetter) *MockSubRepoPermissionsGetter {
	return &MockSubRepoPermissionsGetter{
		GetByUserFunc: &SubRepoPermissionsGetterGetByUserFunc{
			defaultHook: i.GetByUser,
		},
	}
}

// SubRepoPermissionsGetterGetByUserFunc describes the behavior when the
// GetByUser method of the parent MockSubRepoPermissionsGetter instance is
// invoked.
type SubRepoPermissionsGetterGetByUserFunc struct {
	defaultHook func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error)
	hooks       []func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error)
	history     []SubRepoPermissionsGetterGetByUserFuncCall
	mutex       sync.Mutex
}

// GetByUser delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockSubRepoPermissionsGetter) GetByUser(v0 context.Context, v1 int32) (map[api.RepoName]SubRepoPermissions, error) {
	r0, r1 := m.GetByUserFunc.nextHook()(v0, v1)
	m.GetByUserFunc.appendCall(SubRepoPermissionsGetterGetByUserFuncCall{v0, v1, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the GetByUser method of
// the parent MockSubRepoPermissionsGetter instance is invoked and the hook
// queue is empty.
func (f *SubRepoPermissionsGetterGetByUserFunc) SetDefaultHook(hook func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// GetByUser method of the parent MockSubRepoPermissionsGetter instance
// invokes the hook at the front of the queue and discards it. After the
// queue is empty, the default hook function is invoked for any future
// action.
func (f *SubRepoPermissionsGetterGetByUserFunc) PushHook(hook func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *SubRepoPermissionsGetterGetByUserFunc) SetDefaultReturn(r0 map[api.RepoName]SubRepoPermissions, r1 error) {
	f.SetDefaultHook(func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *SubRepoPermissionsGetterGetByUserFunc) PushReturn(r0 map[api.RepoName]SubRepoPermissions, r1 error) {
	f.PushHook(func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error) {
		return r0, r1
	})
}

func (f *SubRepoPermissionsGetterGetByUserFunc) nextHook() func(context.Context, int32) (map[api.RepoName]SubRepoPermissions, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *SubRepoPermissionsGetterGetByUserFunc) appendCall(r0 SubRepoPermissionsGetterGetByUserFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of SubRepoPermissionsGetterGetByUserFuncCall
// objects describing the invocations of this function.
func (f *SubRepoPermissionsGetterGetByUserFunc) History() []SubRepoPermissionsGetterGetByUserFuncCall {
	f.mutex.Lock()
	history := make([]SubRepoPermissionsGetterGetByUserFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// SubRepoPermissionsGetterGetByUserFuncCall is an object that describes an
// invocation of method GetByUser on an instance of
// MockSubRepoPermissionsGetter.
type SubRepoPermissionsGetterGetByUserFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int32
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 map[api.RepoName]SubRepoPermissions
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c SubRepoPermissionsGetterGetByUserFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c SubRepoPermissionsGetterGetByUserFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}
