// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/pkg/mock/go/context (interfaces: Context)

// Package context is a generated GoMock package.
package context

import (
	reflect "reflect"
	time "time"

	gomock "go.uber.org/mock/gomock"
)

// MockContext is a mock of Context interface.
type MockContext struct {
	ctrl     *gomock.Controller
	recorder *MockContextMockRecorder
}

// MockContextMockRecorder is the mock recorder for MockContext.
type MockContextMockRecorder struct {
	mock *MockContext
}

// NewMockContext creates a new mock instance.
func NewMockContext(ctrl *gomock.Controller) *MockContext {
	mock := &MockContext{ctrl: ctrl}
	mock.recorder = &MockContextMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockContext) EXPECT() *MockContextMockRecorder {
	return m.recorder
}

// Deadline mocks base method.
func (m *MockContext) Deadline() (time.Time, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Deadline")
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// Deadline indicates an expected call of Deadline.
func (mr *MockContextMockRecorder) Deadline() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Deadline", reflect.TypeOf((*MockContext)(nil).Deadline))
}

// Done mocks base method.
func (m *MockContext) Done() <-chan struct{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Done")
	ret0, _ := ret[0].(<-chan struct{})
	return ret0
}

// Done indicates an expected call of Done.
func (mr *MockContextMockRecorder) Done() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Done", reflect.TypeOf((*MockContext)(nil).Done))
}

// Err mocks base method.
func (m *MockContext) Err() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Err")
	ret0, _ := ret[0].(error)
	return ret0
}

// Err indicates an expected call of Err.
func (mr *MockContextMockRecorder) Err() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Err", reflect.TypeOf((*MockContext)(nil).Err))
}

// Value mocks base method.
func (m *MockContext) Value(arg0 interface{}) interface{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Value", arg0)
	ret0, _ := ret[0].(interface{})
	return ret0
}

// Value indicates an expected call of Value.
func (mr *MockContextMockRecorder) Value(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Value", reflect.TypeOf((*MockContext)(nil).Value), arg0)
}
