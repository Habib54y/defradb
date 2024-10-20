// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	client "github.com/sourcenetwork/defradb/client"

	mock "github.com/stretchr/testify/mock"
)

// EncodedDocument is an autogenerated mock type for the EncodedDocument type
type EncodedDocument struct {
	mock.Mock
}

type EncodedDocument_Expecter struct {
	mock *mock.Mock
}

func (_m *EncodedDocument) EXPECT() *EncodedDocument_Expecter {
	return &EncodedDocument_Expecter{mock: &_m.Mock}
}

// ID provides a mock function with given fields:
func (_m *EncodedDocument) ID() []byte {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ID")
	}

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0
}

// EncodedDocument_ID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ID'
type EncodedDocument_ID_Call struct {
	*mock.Call
}

// ID is a helper method to define mock.On call
func (_e *EncodedDocument_Expecter) ID() *EncodedDocument_ID_Call {
	return &EncodedDocument_ID_Call{Call: _e.mock.On("ID")}
}

func (_c *EncodedDocument_ID_Call) Run(run func()) *EncodedDocument_ID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *EncodedDocument_ID_Call) Return(_a0 []byte) *EncodedDocument_ID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *EncodedDocument_ID_Call) RunAndReturn(run func() []byte) *EncodedDocument_ID_Call {
	_c.Call.Return(run)
	return _c
}

// Properties provides a mock function with given fields: onlyFilterProps
func (_m *EncodedDocument) Properties(onlyFilterProps bool) (map[client.FieldDefinition]interface{}, error) {
	ret := _m.Called(onlyFilterProps)

	if len(ret) == 0 {
		panic("no return value specified for Properties")
	}

	var r0 map[client.FieldDefinition]interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(bool) (map[client.FieldDefinition]interface{}, error)); ok {
		return rf(onlyFilterProps)
	}
	if rf, ok := ret.Get(0).(func(bool) map[client.FieldDefinition]interface{}); ok {
		r0 = rf(onlyFilterProps)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[client.FieldDefinition]interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(bool) error); ok {
		r1 = rf(onlyFilterProps)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// EncodedDocument_Properties_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Properties'
type EncodedDocument_Properties_Call struct {
	*mock.Call
}

// Properties is a helper method to define mock.On call
//   - onlyFilterProps bool
func (_e *EncodedDocument_Expecter) Properties(onlyFilterProps interface{}) *EncodedDocument_Properties_Call {
	return &EncodedDocument_Properties_Call{Call: _e.mock.On("Properties", onlyFilterProps)}
}

func (_c *EncodedDocument_Properties_Call) Run(run func(onlyFilterProps bool)) *EncodedDocument_Properties_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(bool))
	})
	return _c
}

func (_c *EncodedDocument_Properties_Call) Return(_a0 map[client.FieldDefinition]interface{}, _a1 error) *EncodedDocument_Properties_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *EncodedDocument_Properties_Call) RunAndReturn(run func(bool) (map[client.FieldDefinition]interface{}, error)) *EncodedDocument_Properties_Call {
	_c.Call.Return(run)
	return _c
}

// Reset provides a mock function with given fields:
func (_m *EncodedDocument) Reset() {
	_m.Called()
}

// EncodedDocument_Reset_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Reset'
type EncodedDocument_Reset_Call struct {
	*mock.Call
}

// Reset is a helper method to define mock.On call
func (_e *EncodedDocument_Expecter) Reset() *EncodedDocument_Reset_Call {
	return &EncodedDocument_Reset_Call{Call: _e.mock.On("Reset")}
}

func (_c *EncodedDocument_Reset_Call) Run(run func()) *EncodedDocument_Reset_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *EncodedDocument_Reset_Call) Return() *EncodedDocument_Reset_Call {
	_c.Call.Return()
	return _c
}

func (_c *EncodedDocument_Reset_Call) RunAndReturn(run func()) *EncodedDocument_Reset_Call {
	_c.Call.Return(run)
	return _c
}

// SchemaVersionID provides a mock function with given fields:
func (_m *EncodedDocument) SchemaVersionID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for SchemaVersionID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// EncodedDocument_SchemaVersionID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SchemaVersionID'
type EncodedDocument_SchemaVersionID_Call struct {
	*mock.Call
}

// SchemaVersionID is a helper method to define mock.On call
func (_e *EncodedDocument_Expecter) SchemaVersionID() *EncodedDocument_SchemaVersionID_Call {
	return &EncodedDocument_SchemaVersionID_Call{Call: _e.mock.On("SchemaVersionID")}
}

func (_c *EncodedDocument_SchemaVersionID_Call) Run(run func()) *EncodedDocument_SchemaVersionID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *EncodedDocument_SchemaVersionID_Call) Return(_a0 string) *EncodedDocument_SchemaVersionID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *EncodedDocument_SchemaVersionID_Call) RunAndReturn(run func() string) *EncodedDocument_SchemaVersionID_Call {
	_c.Call.Return(run)
	return _c
}

// Status provides a mock function with given fields:
func (_m *EncodedDocument) Status() client.DocumentStatus {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Status")
	}

	var r0 client.DocumentStatus
	if rf, ok := ret.Get(0).(func() client.DocumentStatus); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(client.DocumentStatus)
	}

	return r0
}

// EncodedDocument_Status_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Status'
type EncodedDocument_Status_Call struct {
	*mock.Call
}

// Status is a helper method to define mock.On call
func (_e *EncodedDocument_Expecter) Status() *EncodedDocument_Status_Call {
	return &EncodedDocument_Status_Call{Call: _e.mock.On("Status")}
}

func (_c *EncodedDocument_Status_Call) Run(run func()) *EncodedDocument_Status_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *EncodedDocument_Status_Call) Return(_a0 client.DocumentStatus) *EncodedDocument_Status_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *EncodedDocument_Status_Call) RunAndReturn(run func() client.DocumentStatus) *EncodedDocument_Status_Call {
	_c.Call.Return(run)
	return _c
}

// NewEncodedDocument creates a new instance of EncodedDocument. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewEncodedDocument(t interface {
	mock.TestingT
	Cleanup(func())
}) *EncodedDocument {
	mock := &EncodedDocument{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}