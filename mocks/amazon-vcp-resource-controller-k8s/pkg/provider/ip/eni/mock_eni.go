// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/aws/amazon-vpc-resource-controller-k8s/pkg/provider/ip/eni (interfaces: ENIManager)

// Package mock_eni is a generated GoMock package.
package mock_eni

import (
	api "github.com/aws/amazon-vpc-resource-controller-k8s/pkg/aws/ec2/api"
	logr "github.com/go-logr/logr"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockENIManager is a mock of ENIManager interface
type MockENIManager struct {
	ctrl     *gomock.Controller
	recorder *MockENIManagerMockRecorder
}

// MockENIManagerMockRecorder is the mock recorder for MockENIManager
type MockENIManagerMockRecorder struct {
	mock *MockENIManager
}

// NewMockENIManager creates a new mock instance
func NewMockENIManager(ctrl *gomock.Controller) *MockENIManager {
	mock := &MockENIManager{ctrl: ctrl}
	mock.recorder = &MockENIManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockENIManager) EXPECT() *MockENIManagerMockRecorder {
	return m.recorder
}

// CreateIPV4Address mocks base method
func (m *MockENIManager) CreateIPV4Address(arg0 int, arg1 api.EC2APIHelper, arg2 logr.Logger) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateIPV4Address", arg0, arg1, arg2)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateIPV4Address indicates an expected call of CreateIPV4Address
func (mr *MockENIManagerMockRecorder) CreateIPV4Address(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateIPV4Address", reflect.TypeOf((*MockENIManager)(nil).CreateIPV4Address), arg0, arg1, arg2)
}

// DeleteIPV4Address mocks base method
func (m *MockENIManager) DeleteIPV4Address(arg0 []string, arg1 api.EC2APIHelper, arg2 logr.Logger) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteIPV4Address", arg0, arg1, arg2)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteIPV4Address indicates an expected call of DeleteIPV4Address
func (mr *MockENIManagerMockRecorder) DeleteIPV4Address(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteIPV4Address", reflect.TypeOf((*MockENIManager)(nil).DeleteIPV4Address), arg0, arg1, arg2)
}

// InitResources mocks base method
func (m *MockENIManager) InitResources(arg0 api.EC2APIHelper) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InitResources", arg0)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// InitResources indicates an expected call of InitResources
func (mr *MockENIManagerMockRecorder) InitResources(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitResources", reflect.TypeOf((*MockENIManager)(nil).InitResources), arg0)
}
