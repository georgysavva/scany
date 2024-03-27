package dbscan

import (
	"fmt"
	"reflect"
)

type startScannerFunc func(rs *RowScanner, dstValue reflect.Value) error

//go:generate mockery --name startScannerFunc --filename mock_test.go --inpackage

// RowScanner embraces Rows and exposes the Scan method
// that allows scanning data from the current row into the destination.
// The first time the Scan method is called
// it parses the destination type via reflection and caches all required information for further scans.
// Due to this caching mechanism, it's not allowed to call Scan for destinations of different types,
// the behavior is unknown in that case.
// RowScanner doesn't proceed to the next row nor close them, it should be done by the client code.
//
// The main benefit of using this type directly
// is that you can instantiate a RowScanner and manually iterate over the rows
// and control how data is scanned from each row.
// This can be beneficial if the result set is large
// and you don't want to allocate a slice for all rows at once
// as it would be done in ScanAll.
//
// ScanOne and ScanAll both use RowScanner type internally.
type RowScanner struct {
	api                *API
	rows               Rows
	columns            []string
	columnToFieldIndex map[string][]int
	mapElementType     reflect.Type
	started            bool
	scanFn             func(dstVal reflect.Value) error
	start              startScannerFunc
}

// NewRowScanner is a package-level helper function that uses the DefaultAPI object.
// See API.NewRowScanner for details.
func NewRowScanner(rows Rows) *RowScanner {
	return DefaultAPI.NewRowScanner(rows)
}

// NewRowScanner returns a new instance of the RowScanner.
func (api *API) NewRowScanner(rows Rows) *RowScanner {
	return &RowScanner{
		api:   api,
		rows:  rows,
		start: startScanner,
	}
}

// Scan scans data from the current row into the destination.
// On the first call it caches expensive reflection work and uses it the future calls.
// See RowScanner for details.
func (rs *RowScanner) Scan(dst interface{}) error {
	dstVal, err := parseDestination(dst)
	if err != nil {
		return fmt.Errorf("parsing destination: %w", err)
	}
	if err := rs.doScan(dstVal); err != nil {
		return fmt.Errorf("doing scan: %w", err)
	}
	return nil
}

func (rs *RowScanner) doScan(dstValue reflect.Value) error {
	if !rs.started {
		if err := rs.start(rs, dstValue); err != nil {
			return fmt.Errorf("starting: %w", err)
		}
		rs.started = true
	}
	if err := rs.scanFn(dstValue); err != nil {
		return fmt.Errorf("scanFn: %w", err)
	}
	return nil
}

func startScanner(rs *RowScanner, dstValue reflect.Value) error {
	var err error
	rs.columns, err = rs.rows.Columns()
	if err != nil {
		return fmt.Errorf("scany: get rows columns: %w", err)
	}
	if err := rs.ensureDistinctColumns(); err != nil {
		return fmt.Errorf("duplicate columns: %w", err)
	}
	dstKind := dstValue.Kind()
	dstType := dstValue.Type()
	isScannable := rs.api.isScannableType(dstType)
	if isScannable && len(rs.columns) == 1 {
		rs.scanFn = rs.scanPrimitive
		return nil
	}

	if dstKind == reflect.Struct {
		rs.columnToFieldIndex = rs.api.getColumnToFieldIndexMap(dstType)
		rs.scanFn = rs.scanStruct
		return nil
	}

	if dstKind == reflect.Map {
		if dstType.Key().Kind() != reflect.String {
			return fmt.Errorf(
				"scany: invalid type %v: map must have string key, got: %v",
				dstType, dstType.Key(),
			)
		}
		rs.mapElementType = dstType.Elem()
		rs.scanFn = rs.scanMap
		return nil
	}

	if len(rs.columns) == 1 {
		rs.scanFn = rs.scanPrimitive
		return nil
	}
	return fmt.Errorf(
		"scany: to scan into a primitive type, columns number must be exactly 1, got: %d",
		len(rs.columns),
	)
}

type noOpScanType struct{}

func (*noOpScanType) Scan(value interface{}) error {
	return nil
}

func (rs *RowScanner) scanStruct(structValue reflect.Value) error {
	scans := make([]interface{}, len(rs.columns))
	for i, column := range rs.columns {
		fieldIndex, ok := rs.columnToFieldIndex[column]
		if !ok {
			if rs.api.allowUnknownColumns {
				var tmp noOpScanType
				scans[i] = &tmp
				continue
			}
			return fmt.Errorf(
				"scany: column: '%s': no corresponding field found, or it's unexported in %v",
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
	if err := rs.rows.Scan(scans...); err != nil {
		return fmt.Errorf("scany: scan row into struct fields: %w", err)
	}
	return nil
}

func (rs *RowScanner) scanMap(mapValue reflect.Value) error {
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapValue.Type()))
	}

	scans := make([]interface{}, len(rs.columns))
	values := make([]reflect.Value, len(rs.columns))
	for i := range rs.columns {
		valuePtr := reflect.New(rs.mapElementType)
		scans[i] = valuePtr.Interface()
		values[i] = valuePtr.Elem()
	}
	if err := rs.rows.Scan(scans...); err != nil {
		return fmt.Errorf("scany: scan rows into map: %w", err)
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
	if err := rs.rows.Scan(value.Addr().Interface()); err != nil {
		return fmt.Errorf("scany: scan row value into a primitive type: %w", err)
	}
	return nil
}

func (rs *RowScanner) ensureDistinctColumns() error {
	seen := make(map[string]struct{}, len(rs.columns))
	for _, column := range rs.columns {
		if _, ok := seen[column]; ok {
			return fmt.Errorf("scany: rows contain a duplicate column '%s'", column)
		}
		seen[column] = struct{}{}
	}
	return nil
}
