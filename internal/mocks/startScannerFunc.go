package mocks

import (
	reflect "reflect"

	mock "github.com/stretchr/testify/mock"

	dbscan2 "github.com/georgysavva/dbscan/dbscan"
)

type StartScannerFunc struct {
	mock.Mock
}

func (_m *StartScannerFunc) Execute(rs *dbscan2.RowScanner, dstValue reflect.Value) error {
	ret := _m.Called(rs, dstValue)

	var r0 error
	if rf, ok := ret.Get(0).(func(*dbscan2.RowScanner, reflect.Value) error); ok {
		r0 = rf(rs, dstValue)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
