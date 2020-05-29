package pgxquery

import (
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"reflect"
)

type destinationMeta struct {
	columnToFieldIndex map[string][]int
	sliceElementType   reflect.Type
	sliceElementByPtr  bool
	mapElementType     reflect.Type
}

func parseDestination(dst interface{}, sliceExpected bool) (reflect.Value, *destinationMeta, error) {
	dstVal := reflect.ValueOf(dst)

	if !dstVal.IsValid() || (dstVal.Kind() == reflect.Ptr && dstVal.IsNil()) {
		return reflect.Value{}, nil, errors.Errorf("destinationMeta must be a non nil pointer")
	}
	if dstVal.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, errors.Errorf("destinationMeta must be a pointer, got: %v", dstVal.Type())
	}

	dstElemVal := dstVal.Elem()

	meta := &destinationMeta{}
	baseType := dstElemVal.Type()
	if sliceExpected {
		if dstElemVal.Kind() != reflect.Slice {
			return reflect.Value{}, nil, errors.Errorf(
				"destinationMeta must be a pointer to a slice, got: %v", dstVal.Type(),
			)
		}
		meta.sliceElementType = dstElemVal.Type().Elem()

		// If it's a slice of structs or maps,
		// we handle them the same way and dereference pointers to values,
		// because eventually we works with fields or keys.
		// But if it's a slice of primitive type e.g. or []string or []*string,
		// we must leave and pass elements as is to Rows.Scan().
		if meta.sliceElementType.Kind() == reflect.Ptr {
			if meta.sliceElementType.Elem().Kind() == reflect.Struct ||
				meta.sliceElementType.Elem().Kind() == reflect.Map {

				meta.sliceElementByPtr = true
				meta.sliceElementType = meta.sliceElementType.Elem()
			}
		}
		baseType = meta.sliceElementType
	}

	if baseType.Kind() == reflect.Struct {
		var err error
		meta.columnToFieldIndex, err = getColumnToFieldIndexMap(baseType)
		if err != nil {
			return reflect.Value{}, nil, errors.WithStack(err)
		}
	} else if baseType.Kind() == reflect.Map {
		if baseType.Key().Kind() != reflect.String {
			return reflect.Value{}, nil, errors.Errorf(
				"invalid type %v: map must have string key, got: %v",
				baseType, baseType.Key(),
			)
		}
		meta.mapElementType = baseType.Elem()
	}

	return dstElemVal, meta, nil
}

func (d *destinationMeta) fillSliceElement(sliceValue reflect.Value, rows pgx.Rows) error {
	elemVal := reflect.New(d.sliceElementType).Elem()
	if err := d.fill(elemVal, rows); err != nil {
		return errors.WithStack(err)
	}
	if d.sliceElementByPtr {
		elemVal = elemVal.Addr()
	}
	sliceValue.Set(reflect.Append(sliceValue, elemVal))
	return nil
}

func (d *destinationMeta) fill(dstValue reflect.Value, rows pgx.Rows) error {
	var err error
	if dstValue.Kind() == reflect.Struct {
		err = d.fillStruct(dstValue, rows)
	} else if dstValue.Kind() == reflect.Map {
		err = d.fillMap(dstValue, rows)
	} else {
		err = fillPrimitive(dstValue, rows)
	}
	return errors.WithStack(err)
}

func (d *destinationMeta) fillStruct(structValue reflect.Value, rows pgx.Rows) error {
	if err := ensureDistinctColumns(rows); err != nil {
		return errors.WithStack(err)
	}

	scans := make([]interface{}, len(rows.FieldDescriptions()))
	for i, columnDesc := range rows.FieldDescriptions() {
		column := string(columnDesc.Name)
		fieldIndex, ok := d.columnToFieldIndex[column]
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
		return errors.Wrap(err, "fill row into struct fields")
	}
	return nil
}

func (d *destinationMeta) fillMap(mapValue reflect.Value, rows pgx.Rows) error {
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
		if !value.Type().ConvertibleTo(d.mapElementType) {
			return errors.Errorf(
				"Column '%s' value of type %v can'be set into %v",
				column, value.Type(), mapValue.Type(),
			)
		}
		mapValue.SetMapIndex(key, value.Convert(d.mapElementType))
	}

	return nil
}

func fillPrimitive(value reflect.Value, rows pgx.Rows) error {
	columnsNumber := len(rows.FieldDescriptions())
	if columnsNumber != 1 {
		return errors.Errorf(
			"to fill into a primitive type, columns number must be exactly 1, got: %d",
			columnsNumber,
		)
	}
	if err := rows.Scan(value.Addr().Interface()); err != nil {
		return errors.Wrap(err, "fill row value into primitive type")
	}
	return nil
}

func initializeNested(structValue reflect.Value, fieldIndex []int) {
	i := fieldIndex[0]
	field := structValue.Field(i)

	// Create a new instance of a struct and set it to field,
	// if field is a nil pointer to a struct.
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct && field.IsNil() {
		field.Set(reflect.New(field.Type().Elem()))
	}
	if len(fieldIndex) > 1 {
		initializeNested(reflect.Indirect(field), fieldIndex[1:])
	}
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
