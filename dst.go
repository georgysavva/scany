package pgxquery

import (
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"reflect"
	"regexp"
	"strings"
)

type destination struct {
	dstValue           reflect.Value
	exactlyOneRow      bool
	columnToFieldIndex map[string][]int
	sliceElementType   reflect.Type
	sliceElementByPtr  bool
	mapElementType     reflect.Type
}

func parseDestination(dst interface{}, exactlyOneRow bool) (*destination, error) {
	dstVal := reflect.ValueOf(dst)

	if !dstVal.IsValid() || (dstVal.Kind() == reflect.Ptr && dstVal.IsNil()) {
		return nil, errors.Errorf("destination must be a non nil pointer")
	}
	if dstVal.Kind() != reflect.Ptr {
		return nil, errors.Errorf("destination must be a pointer, got: %v", dstVal.Type())
	}

	dstElemVal := dstVal.Elem()

	d := &destination{
		dstValue:      dstElemVal,
		exactlyOneRow: exactlyOneRow,
	}
	baseType := dstElemVal.Type()
	if !exactlyOneRow {
		if dstElemVal.Kind() != reflect.Slice {
			return nil, errors.Errorf(
				"destination must be a pointer to a slice, got: %v", dstVal.Type(),
			)
		}
		d.sliceElementType = dstElemVal.Type().Elem()

		// If it's a slice of structs or maps,
		// we handle them the same way and dereference pointers to values,
		// because eventually we works with fields or keys.
		// But if it's a slice of primitive type e.g. or []string or []*string,
		// we must leave and pass elements as is to Rows.Scan().
		if d.sliceElementType.Kind() == reflect.Ptr {
			if d.sliceElementType.Elem().Kind() == reflect.Struct ||
				d.sliceElementType.Elem().Kind() == reflect.Map {

				d.sliceElementByPtr = true
				d.sliceElementType = d.sliceElementType.Elem()
			}
		}
		baseType = d.sliceElementType
	}

	if baseType.Kind() == reflect.Struct {
		var err error
		d.columnToFieldIndex, err = getColumnToFieldIndexMap(baseType)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else if baseType.Kind() == reflect.Map {
		if baseType.Key().Kind() != reflect.String {
			return nil, errors.Errorf(
				"invalid type %v: map must have string key, got: %v",
				baseType, baseType.Key(),
			)
		}
		d.mapElementType = baseType.Elem()
	}

	return d, nil
}

func (d *destination) scanRows(rows pgx.Rows) (int, error) {
	if !d.exactlyOneRow {
		// Make sure that slice is empty.
		d.dstValue.Set(d.dstValue.Slice(0, 0))
	}
	var rowsAffected int
	for rows.Next() {
		var err error
		if d.exactlyOneRow {
			err = d.fillElement(d.dstValue, rows)
		} else {
			err = d.fillSlice(rows)
		}
		if err != nil {
			return 0, errors.WithStack(err)
		}
		rowsAffected++
	}
	return rowsAffected, nil
}

func (d *destination) fillSlice(rows pgx.Rows) error {
	elemVal := reflect.New(d.sliceElementType).Elem()
	if err := d.fillElement(elemVal, rows); err != nil {
		return errors.WithStack(err)
	}
	if d.sliceElementByPtr {
		elemVal = elemVal.Addr()
	}
	d.dstValue.Set(reflect.Append(d.dstValue, elemVal))
	return nil
}

func (d *destination) fillElement(elementValue reflect.Value, rows pgx.Rows) error {
	var err error
	if elementValue.Kind() == reflect.Struct {
		err = d.fillStruct(elementValue, rows)
	} else if elementValue.Kind() == reflect.Map {
		err = d.fillMap(elementValue, rows)
	} else {
		err = fillPrimitive(elementValue, rows)
	}
	return errors.WithStack(err)
}

func (d *destination) fillStruct(structValue reflect.Value, rows pgx.Rows) error {
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

func (d *destination) fillMap(mapValue reflect.Value, rows pgx.Rows) error {
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

var dbStructTagKey = "db"

func getColumnToFieldIndexMap(structType reflect.Type) (map[string][]int, error) {
	result := make(map[string][]int, structType.NumField())

	setColumn := func(column string, index []int) error {
		if otherIndex, ok := result[column]; ok {
			return errors.Errorf(
				"Column must have exactly one field pointing to it; "+
					"found 2 fields with indexes %d and %d pointing to '%s' in %v",
				otherIndex, index, column, structType,
			)
		}
		result[column] = index
		return nil
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		if field.PkgPath != "" {
			// Field is unexported, skip it.
			continue
		}

		dbTag := field.Tag.Get(dbStructTagKey)

		if dbTag == "-" {
			// Field is ignored, skip it.
			continue
		}

		if field.Anonymous {
			childType := field.Type
			if field.Type.Kind() == reflect.Ptr {
				childType = field.Type.Elem()
			}
			if childType.Kind() == reflect.Struct {
				// Field is embedded struct or pointer to struct.
				childMap, err := getColumnToFieldIndexMap(childType)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				for childColumn, childIndex := range childMap {
					column := childColumn
					// If "db" tag is present for embedded struct
					// use it with "." to prefix all column from the embedded struct.
					// the default behaviour is to propagate columns as is.
					if dbTag != "" {
						column = dbTag + "." + column
					}
					index := append(field.Index, childIndex...)
					if err := setColumn(column, index); err != nil {
						return nil, errors.WithStack(err)
					}
				}
				continue
			}
		}

		column := dbTag
		if dbTag == "" {
			column = toSnakeCase(field.Name)
		}
		if err := setColumn(column, field.Index); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	return result, nil
}

var matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCapRe = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
