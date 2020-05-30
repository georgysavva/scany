package pgxquery

import (
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"reflect"
)

type Rows struct {
	pgx.Rows
	started            bool
	columnToFieldIndex map[string][]int
	sliceElementType   reflect.Type
	sliceElementByPtr  bool
	mapElementType     reflect.Type
}

func WrapRows(rows pgx.Rows) *Rows {
	return &Rows{Rows: rows}
}

func (r *Rows) Scanx(dst interface{}) error {
	dstVal, err := parseDestination(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := r.doScan(dstVal); err != nil {
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

func (r *Rows) doScan(dstValue reflect.Value) error {
	var err error
	if dstValue.Kind() == reflect.Struct {
		err = r.scanStruct(dstValue)
	} else if dstValue.Kind() == reflect.Map {
		err = r.scanMap(dstValue)
	} else {
		err = r.scanPrimitive(dstValue)
	}
	return errors.WithStack(err)
}

func wrapRowsForSliceScan(rows pgx.Rows, sliceType reflect.Type) *Rows {
	var sliceElementByPtr bool
	sliceElementType := sliceType.Elem()

	// If it'r a slice of structs or maps,
	// we handle them the same way and dereference pointers to values,
	// because eventually we works with fields or keys.
	// But if it'r a slice of primitive type e.g. or []string or []*string,
	// we must leave and pass elements as is to Rows.Scan().
	if sliceElementType.Kind() == reflect.Ptr {
		if sliceElementType.Elem().Kind() == reflect.Struct ||
			sliceElementType.Elem().Kind() == reflect.Map {

			sliceElementByPtr = true
			sliceElementType = sliceElementType.Elem()
		}
	}
	r := &Rows{
		Rows:              rows,
		sliceElementType:  sliceElementType,
		sliceElementByPtr: sliceElementByPtr,
	}
	return r
}

func (r *Rows) scanSliceElement(sliceValue reflect.Value) error {
	elemVal := reflect.New(r.sliceElementType).Elem()
	if err := r.doScan(elemVal); err != nil {
		return errors.WithStack(err)
	}
	if r.sliceElementByPtr {
		elemVal = elemVal.Addr()
	}
	sliceValue.Set(reflect.Append(sliceValue, elemVal))
	return nil
}

func (r *Rows) scanStruct(structValue reflect.Value) error {
	if !r.started {
		if err := r.ensureDistinctColumns(); err != nil {
			return errors.WithStack(err)
		}
		var err error
		r.columnToFieldIndex, err = getColumnToFieldIndexMap(structValue.Type())
		if err != nil {
			return errors.WithStack(err)
		}
		r.started = true
	}

	scans := make([]interface{}, len(r.Rows.FieldDescriptions()))
	for i, columnDesc := range r.Rows.FieldDescriptions() {
		column := string(columnDesc.Name)
		fieldIndex, ok := r.columnToFieldIndex[column]
		if !ok {
			return errors.Errorf(
				"column: '%r': no corresponding field found or it'r unexported in %v",
				column, structValue.Type(),
			)
		}
		// Struct may contain embedded structs by ptr that defaults to nil.
		// In order to doScan values into a nested field,
		// we need to initialize all nil structs on its way.
		initializeNested(structValue, fieldIndex)

		fieldVal := structValue.FieldByIndex(fieldIndex)
		if !fieldVal.Addr().CanInterface() {
			return errors.Errorf(
				"column: '%r': corresponding field with index %d is invalid or can't be set in %v",
				column, fieldIndex, structValue.Type(),
			)
		}
		scans[i] = fieldVal.Addr().Interface()
	}
	if err := r.Rows.Scan(scans...); err != nil {
		return errors.Wrap(err, "doScan row into struct fields")
	}
	return nil
}

func (r *Rows) scanMap(mapValue reflect.Value) error {
	if !r.started {
		mapType := mapValue.Type()
		if mapType.Key().Kind() != reflect.String {
			return errors.Errorf(
				"invalid type %v: map must have string key, got: %v",
				mapType, mapType.Key(),
			)
		}
		r.mapElementType = mapType.Elem()
		r.started = true
	}
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapValue.Type()))
	}

	values, err := r.Rows.Values()
	if err != nil {
		return errors.Wrap(err, "get row values for map")
	}

	if err := r.ensureDistinctColumns(); err != nil {
		return errors.WithStack(err)
	}

	for i, columnDesc := range r.Rows.FieldDescriptions() {
		column := string(columnDesc.Name)
		key := reflect.ValueOf(column)
		value := reflect.ValueOf(values[i])

		// If value type is different compared to map element type, try to convert it,
		// if they aren't convertible there is nothing we can do to set it.
		if !value.Type().ConvertibleTo(r.mapElementType) {
			return errors.Errorf(
				"Column '%r' value of type %v can'be set into %v",
				column, value.Type(), mapValue.Type(),
			)
		}
		mapValue.SetMapIndex(key, value.Convert(r.mapElementType))
	}

	return nil
}

func (r *Rows) scanPrimitive(value reflect.Value) error {
	if !r.started {
		columnsNumber := len(r.Rows.FieldDescriptions())
		if columnsNumber != 1 {
			return errors.Errorf(
				"to doScan into a primitive type, columns number must be exactly 1, got: %d",
				columnsNumber,
			)
		}
		r.started = true
	}
	if err := r.Rows.Scan(value.Addr().Interface()); err != nil {
		return errors.Wrap(err, "doScan row value into primitive type")
	}
	return nil
}

func (r *Rows) ensureDistinctColumns() error {
	seen := make(map[string]struct{}, len(r.Rows.FieldDescriptions()))
	for _, columnDesc := range r.Rows.FieldDescriptions() {
		column := string(columnDesc.Name)
		if _, ok := seen[column]; ok {
			return errors.Errorf("row contains duplicated column '%s'", column)
		}
		seen[column] = struct{}{}
	}
	return nil
}
