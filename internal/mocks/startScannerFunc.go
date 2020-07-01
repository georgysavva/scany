package mocks

import (
	reflect "reflect"

	mock "github.com/stretchr/testify/mock"

	dbscan "github.com/georgysavva/scany/dbscan"
)

type StartScannerFunc struct {
	mock.Mock
}

func (_m *StartScannerFunc) Execute(rs *dbscan.RowScanner, dstValue reflect.Value) error {
	ret := _m.Called(rs, dstValue)

	var r0 error
	if rf, ok := ret.Get(0).(func(*dbscan.RowScanner, reflect.Value) error); ok {
		r0 = rf(rs, dstValue)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
