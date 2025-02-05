// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/extensions/pkg/util (interfaces: ShootClients)

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	kubernetes "github.com/gardener/gardener/pkg/client/kubernetes"
	gomock "go.uber.org/mock/gomock"
	version "k8s.io/apimachinery/pkg/version"
	kubernetes0 "k8s.io/client-go/kubernetes"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockShootClients is a mock of ShootClients interface.
type MockShootClients struct {
	ctrl     *gomock.Controller
	recorder *MockShootClientsMockRecorder
}

// MockShootClientsMockRecorder is the mock recorder for MockShootClients.
type MockShootClientsMockRecorder struct {
	mock *MockShootClients
}

// NewMockShootClients creates a new mock instance.
func NewMockShootClients(ctrl *gomock.Controller) *MockShootClients {
	mock := &MockShootClients{ctrl: ctrl}
	mock.recorder = &MockShootClientsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockShootClients) EXPECT() *MockShootClientsMockRecorder {
	return m.recorder
}

// ChartApplier mocks base method.
func (m *MockShootClients) ChartApplier() kubernetes.ChartApplier {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChartApplier")
	ret0, _ := ret[0].(kubernetes.ChartApplier)
	return ret0
}

// ChartApplier indicates an expected call of ChartApplier.
func (mr *MockShootClientsMockRecorder) ChartApplier() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChartApplier", reflect.TypeOf((*MockShootClients)(nil).ChartApplier))
}

// Client mocks base method.
func (m *MockShootClients) Client() client.Client {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Client")
	ret0, _ := ret[0].(client.Client)
	return ret0
}

// Client indicates an expected call of Client.
func (mr *MockShootClientsMockRecorder) Client() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Client", reflect.TypeOf((*MockShootClients)(nil).Client))
}

// Clientset mocks base method.
func (m *MockShootClients) Clientset() kubernetes0.Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Clientset")
	ret0, _ := ret[0].(kubernetes0.Interface)
	return ret0
}

// Clientset indicates an expected call of Clientset.
func (mr *MockShootClientsMockRecorder) Clientset() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Clientset", reflect.TypeOf((*MockShootClients)(nil).Clientset))
}

// GardenerClientset mocks base method.
func (m *MockShootClients) GardenerClientset() kubernetes.Interface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GardenerClientset")
	ret0, _ := ret[0].(kubernetes.Interface)
	return ret0
}

// GardenerClientset indicates an expected call of GardenerClientset.
func (mr *MockShootClientsMockRecorder) GardenerClientset() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GardenerClientset", reflect.TypeOf((*MockShootClients)(nil).GardenerClientset))
}

// Version mocks base method.
func (m *MockShootClients) Version() *version.Info {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Version")
	ret0, _ := ret[0].(*version.Info)
	return ret0
}

// Version indicates an expected call of Version.
func (mr *MockShootClientsMockRecorder) Version() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Version", reflect.TypeOf((*MockShootClients)(nil).Version))
}
