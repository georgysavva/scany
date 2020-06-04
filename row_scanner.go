package sqlscan

import (
	"reflect"

	"github.com/pkg/errors"
)

type startRowsFunc func(dstValue reflect.Value) error

type RowScanner struct {
	rows               Rows
	columns            []string
	columnToFieldIndex map[string][]int
	sliceElementType   reflect.Type
	sliceElementByPtr  bool
	mapElementType     reflect.Type
	started            bool
	startFn            startRowsFunc
}

func NewRowScanner(rows Rows) *RowScanner {
	r := &RowScanner{rows: rows}
	r.startFn = r.start
	return r
}

func (rs *RowScanner) Scan(dst interface{}) error {
	dstVal, err := parseDestination(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	err = rs.doScan(dstVal)
	return errors.WithStack(err)
}

func parseDestination(dst interface{}) (reflect.Value, error) {
	dstVal := reflect.ValueOf(dst)

	if !dstVal.IsValid() || (dstVal.Kind() == reflect.Ptr && dstVal.IsNil()) {
		return reflect.Value{}, errors.Errorf("destination must be a non nil pointer")
	}
	if dstVal.Kind() != reflect.Ptr {
		return reflect.Value{}, errors.Errorf("destination must be a pointer, got: %v", dstVal.Type())
	}

	dstVal = dstVal.Elem()
	return dstVal, nil
}

func (rs *RowScanner) doScan(dstValue reflect.Value) error {
	if !rs.started {
		if err := rs.startFn(dstValue); err != nil {
			return errors.WithStack(err)
		}
		rs.started = true
	}
	var err error
	if dstValue.Kind() == reflect.Struct {
		err = rs.scanStruct(dstValue)
	} else if dstValue.Kind() == reflect.Map {
		err = rs.scanMap(dstValue)
	} else {
		err = rs.scanPrimitive(dstValue)
	}
	return errors.WithStack(err)
}

func newRowScannerForSliceScan(rows Rows, sliceType reflect.Type) *RowScanner {
	var sliceElementByPtr bool
	sliceElementType := sliceType.Elem()

	// If it's a slice of pointers to structs,
	// we handle it the same way as it would be slice of struct by value
	// and dereference pointers to values,
	// because eventually we works with fields.
	// But if it's a slice of primitive type e.g. or []string or []*string,
	// we must leave and pass elements as is to Rows.Scan().
	if sliceElementType.Kind() == reflect.Ptr {
		if sliceElementType.Elem().Kind() == reflect.Struct {

			sliceElementByPtr = true
			sliceElementType = sliceElementType.Elem()
		}
	}
	rs := NewRowScanner(rows)
	rs.sliceElementType = sliceElementType
	rs.sliceElementByPtr = sliceElementByPtr
	return rs
}

func (rs *RowScanner) start(dstValue reflect.Value) error {
	var err error
	rs.columns, err = rs.rows.Columns()
	if err != nil {
		return errors.Wrap(err, "get columns from rows")
	}
	if err := rs.ensureDistinctColumns(); err != nil {
		return errors.WithStack(err)
	}
	if dstValue.Kind() == reflect.Struct {
		var err error
		rs.columnToFieldIndex, err = getColumnToFieldIndexMap(dstValue.Type())
		return errors.WithStack(err)
	}
	if dstValue.Kind() == reflect.Map {
		mapType := dstValue.Type()
		if mapType.Key().Kind() != reflect.String {
			return errors.Errorf(
				"invalid type %v: map must have string key, got: %v",
				mapType, mapType.Key(),
			)
		}
		rs.mapElementType = mapType.Elem()
		return nil
	}
	// It's the primitive type case.
	columnsNumber := len(rs.columns)
	if columnsNumber != 1 {
		return errors.Errorf(
			"to scan into a primitive type, columns number must be exactly 1, got: %d",
			columnsNumber,
		)
	}
	return nil
}

func (rs *RowScanner) scanSliceElement(sliceValue reflect.Value) error {
	elemVal := reflect.New(rs.sliceElementType).Elem()
	if err := rs.doScan(elemVal); err != nil {
		return errors.WithStack(err)
	}
	if rs.sliceElementByPtr {
		elemVal = elemVal.Addr()
	}
	sliceValue.Set(reflect.Append(sliceValue, elemVal))
	return nil
}

func (rs *RowScanner) scanStruct(structValue reflect.Value) error {
	scans := make([]interface{}, len(rs.columns))
	for i, column := range rs.columns {
		fieldIndex, ok := rs.columnToFieldIndex[column]
		if !ok {
			return errors.Errorf(
				"column: '%s': no corresponding field found or it's unexported in %v",
				column, structValue.Type(),
			)
		}
		// Struct may contain embedded structs by ptr that defaults to nil.
		// In order to scan values into a nested field,
		// we need to initialize all nil structs on its way.
		initializeNested(structValue, fieldIndex)

		fieldVal := structValue.FieldByIndex(fieldIndex)
		scans[i] = fieldVal.Addr().Interface()
	}
	err := rs.rows.Scan(scans...)
	return errors.Wrap(err, "scan row into struct fields")
}

func (rs *RowScanner) scanMap(mapValue reflect.Value) error {
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapValue.Type()))
	}

	scans := make([]interface{}, len(rs.columns))
	values := make([]reflect.Value, len(rs.columns))
	for i := range rs.columns {
		value := reflect.New(rs.mapElementType).Elem()
		scans[i] = value.Addr().Interface()
		values[i] = value
	}
	if err := rs.rows.Scan(scans...); err != nil {
		return errors.Wrap(err, "scan rows into map")
	}
	// We can't set reflect values into destination map before scanning them,
	// because reflect will set a copy, just like regular map behaves,
	// and scan won't modify the map element.
	for i, column := range rs.columns {
		key := reflect.ValueOf(column)
		value := values[i]
		mapValue.SetMapIndex(key, value)
	}
	return nil
}

func (rs *RowScanner) scanPrimitive(value reflect.Value) error {
	err := rs.rows.Scan(value.Addr().Interface())
	return errors.Wrap(err, "scan row value into primitive type")
}

func (rs *RowScanner) ensureDistinctColumns() error {
	seen := make(map[string]struct{}, len(rs.columns))
	for _, column := range rs.columns {
		if _, ok := seen[column]; ok {
			return errors.Errorf("row contains duplicated column '%s'", column)
		}
		seen[column] = struct{}{}
	}
	return nil
}
