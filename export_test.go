package pgxquery

import (
	"reflect"
)

var ParseDestination = parseDestination

func (r *Rows) DoScan(dstValue reflect.Value) error { return r.doScan(dstValue) }
