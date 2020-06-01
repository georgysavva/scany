package pgxscan

import (
	"reflect"
)

var ParseDestination = parseDestination

func (r *Rows) DoScan(dstValue reflect.Value) error { return r.doScan(dstValue) }
func (r *Rows) SetStartFn(f startRowsFunc)          { r.startFn = f }
func (r *Rows) Started() bool                       { return r.started }
