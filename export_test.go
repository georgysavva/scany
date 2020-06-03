package sqlscan

import (
	"reflect"
)

var ParseDestination = parseDestination

func (rs *RowScanner) DoScan(dstValue reflect.Value) error { return rs.doScan(dstValue) }
func (rs *RowScanner) SetStartFn(f startRowsFunc)          { rs.startFn = f }
func (rs *RowScanner) Started() bool                       { return rs.started }
