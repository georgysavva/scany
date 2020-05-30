package pgxquery

import (
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"reflect"
)

type Scanner struct {
	started            bool
	columnToFieldIndex map[string][]int
	sliceElementType   reflect.Type
	sliceElementByPtr  bool
	mapElementType     reflect.Type
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) ScanRow(dst interface{}, rows pgx.Rows) error {
	dstVal, err := parseDestination(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := s.scan(dstVal, rows); err != nil {
		return errors.WithStack(err)
	}
	return nil
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

func (s *Scanner) scan(dstValue reflect.Value, rows pgx.Rows) error {
	var err error
	if dstValue.Kind() == reflect.Struct {
		err = s.scanStruct(dstValue, rows)
	} else if dstValue.Kind() == reflect.Map {
		err = s.scanMap(dstValue, rows)
	} else {
		err = s.scanPrimitive(dstValue, rows)
	}
	return errors.WithStack(err)
}

func newScannerForSlice(sliceType reflect.Type) *Scanner {
	var sliceElementByPtr bool
	sliceElementType := sliceType.Elem()

	// If it's a slice of structs or maps,
	// we handle them the same way and dereference pointers to values,
	// because eventually we works with fields or keys.
	// But if it's a slice of primitive type e.g. or []string or []*string,
	// we must leave and pass elements as is to Rows.Scan().
	if sliceElementType.Kind() == reflect.Ptr {
		if sliceElementType.Elem().Kind() == reflect.Struct ||
			sliceElementType.Elem().Kind() == reflect.Map {

			sliceElementByPtr = true
			sliceElementType = sliceElementType.Elem()
		}
	}
	s := &Scanner{
		sliceElementType:  sliceElementType,
		sliceElementByPtr: sliceElementByPtr,
	}
	return s
}

func (s *Scanner) scanSliceElement(sliceValue reflect.Value, rows pgx.Rows) error {
	elemVal := reflect.New(s.sliceElementType).Elem()
	if err := s.scan(elemVal, rows); err != nil {
		return errors.WithStack(err)
	}
	if s.sliceElementByPtr {
		elemVal = elemVal.Addr()
	}
	sliceValue.Set(reflect.Append(sliceValue, elemVal))
	return nil
}

func (s *Scanner) scanStruct(structValue reflect.Value, rows pgx.Rows) error {
	if !s.started {
		if err := ensureDistinctColumns(rows); err != nil {
			return errors.WithStack(err)
		}
		var err error
		s.columnToFieldIndex, err = getColumnToFieldIndexMap(structValue.Type())
		if err != nil {
			return errors.WithStack(err)
		}
		s.started = true
	}

	scans := make([]interface{}, len(rows.FieldDescriptions()))
	for i, columnDesc := range rows.FieldDescriptions() {
		column := string(columnDesc.Name)
		fieldIndex, ok := s.columnToFieldIndex[column]
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
		if !fieldVal.Addr().CanInterface() {
			return errors.Errorf(
				"column: '%s': corresponding field with index %d is invalid or can't be set in %v",
				column, fieldIndex, structValue.Type(),
			)
		}
		scans[i] = fieldVal.Addr().Interface()
	}
	if err := rows.Scan(scans...); err != nil {
		return errors.Wrap(err, "scan row into struct fields")
	}
	return nil
}

func (s *Scanner) scanMap(mapValue reflect.Value, rows pgx.Rows) error {
	if !s.started {
		mapType := mapValue.Type()
		if mapType.Key().Kind() != reflect.String {
			return errors.Errorf(
				"invalid type %v: map must have string key, got: %v",
				mapType, mapType.Key(),
			)
		}
		s.mapElementType = mapType.Elem()
		s.started = true
	}
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapValue.Type()))
	}

	values, err := rows.Values()
	if err != nil {
		return errors.Wrap(err, "get row values for map")
	}

	if err := ensureDistinctColumns(rows); err != nil {
		return errors.WithStack(err)
	}

	for i, columnDesc := range rows.FieldDescriptions() {
		column := string(columnDesc.Name)
		key := reflect.ValueOf(column)
		value := reflect.ValueOf(values[i])

		// If value type is different compared to map element type, try to convert it,
		// if they aren't convertible there is nothing we can do to set it.
		if !value.Type().ConvertibleTo(s.mapElementType) {
			return errors.Errorf(
				"Column '%s' value of type %v can'be set into %v",
				column, value.Type(), mapValue.Type(),
			)
		}
		mapValue.SetMapIndex(key, value.Convert(s.mapElementType))
	}

	return nil
}

func (s *Scanner) scanPrimitive(value reflect.Value, rows pgx.Rows) error {
	if !s.started {
		columnsNumber := len(rows.FieldDescriptions())
		if columnsNumber != 1 {
			return errors.Errorf(
				"to scan into a primitive type, columns number must be exactly 1, got: %d",
				columnsNumber,
			)
		}
		s.started = true
	}
	if err := rows.Scan(value.Addr().Interface()); err != nil {
		return errors.Wrap(err, "scan row value into primitive type")
	}
	return nil
}

func ensureDistinctColumns(rows pgx.Rows) error {
	seen := make(map[string]struct{}, len(rows.FieldDescriptions()))
	for _, columnDesc := range rows.FieldDescriptions() {
		column := string(columnDesc.Name)
		if _, ok := seen[column]; ok {
			return errors.Errorf("row contains duplicated column '%s'", column)
		}
		seen[column] = struct{}{}
	}
	return nil
}
