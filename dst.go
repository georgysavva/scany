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
	columnToFieldIndex map[string][]int
	sliceElementType   reflect.Type
	sliceElementByPtr  bool
	mapElementType     reflect.Type
}

func parseDst(dst interface{}, exactlyOneRow bool) (reflect.Value, error) {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr {
		return reflect.Value{}, errors.Errorf("destination must be a pointer, got: %v", dstVal.Type())
	}
	dstElemVal := dstVal.Elem()
	if !dstElemVal.IsValid() || !dstElemVal.CanSet() {
		return reflect.Value{}, errors.Errorf("destination must be a valid non nil pointer")
	}

	if !exactlyOneRow {
		if dstElemVal.Kind() != reflect.Slice {
			return reflect.Value{}, errors.Errorf(
				"destination must be a pointer to a slice, got: %v", dstVal.Type(),
			)
		}

	}
	return dstElemVal, nil
}

func fillDestination(dstValue reflect.Value, rows pgx.Rows) (int, error) {
	isSlice := dstValue.Kind() == reflect.Slice
	if isSlice {
		// Make sure that slice is empty.
		dstValue.Set(dstValue.Slice(0, 0))
	}
	dst := &destination{dstValue: dstValue}
	var rowsAffected int
	for rows.Next() {
		var err error
		if isSlice {
			err = dst.fillSlice(rows)
		} else {
			err = dst.fillElement(dstValue, rows)
		}
		if err != nil {
			return 0, errors.WithStack(err)
		}
		rowsAffected++
	}
	return rowsAffected, nil
}

func (d *destination) fillSlice(rows pgx.Rows) error {
	if d.sliceElementType == nil {
		sliceElemType := d.dstValue.Type().Elem()
		if sliceElemType.Kind() == reflect.Ptr {
			d.sliceElementByPtr = true
			sliceElemType = sliceElemType.Elem()
		}
		d.sliceElementType = sliceElemType
	}

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

func (d *destination) fillStruct(elementValue reflect.Value, rows pgx.Rows) error {
	if d.columnToFieldIndex == nil {
		var err error
		d.columnToFieldIndex, err = getColumnToFieldIndexMap(elementValue.Type())
		if err != nil {
			return errors.WithStack(err)
		}
	}

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
				column, elementValue.Type(),
			)
		}
		fieldVal := elementValue.FieldByIndex(fieldIndex)
		if !fieldVal.IsValid() || !fieldVal.CanSet() || !fieldVal.Addr().CanInterface() {
			return errors.Errorf(
				"column: '%s': corresponding field with index %d is invalid or can't be set in %v",
				column, fieldIndex, elementValue.Type(),
			)
		}
		scans[i] = fieldVal.Addr().Interface()
	}
	if err := rows.Scan(scans...); err != nil {
		return errors.Wrap(err, "fillDestination row into struct fields")
	}
	return nil
}

func (d *destination) fillMap(elementValue reflect.Value, rows pgx.Rows) error {
	if d.mapElementType == nil {
		dstType := elementValue.Type()
		if dstType.Key().Kind() != reflect.String {
			return errors.Errorf(
				"invalid element type %v: map must have string key, got: %v",
				dstType, dstType.Key(),
			)
		}
		d.mapElementType = dstType.Elem()
	}

	if elementValue.IsNil() {
		elementValue.Set(reflect.MakeMap(elementValue.Type()))
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
		columnValue := values[i]
		key := reflect.ValueOf(column)
		elem := reflect.ValueOf(columnValue)
		if !elem.Type().ConvertibleTo(d.mapElementType) {
			return errors.Errorf(
				"Column '%s' value of type %v can'be set into %v",
				column, elem.Type(), elementValue.Type(),
			)
		}
		elementValue.SetMapIndex(key, elem.Convert(d.mapElementType))
	}

	return nil
}

func fillPrimitive(elementValue reflect.Value, rows pgx.Rows) error {
	columnsNumber := len(rows.FieldDescriptions())
	if columnsNumber != 1 {
		return errors.Errorf(
			"to fillDestination into a primitive type, columns number must be exactly 1, got: %d",
			columnsNumber,
		)
	}
	if err := rows.Scan(elementValue.Addr().Interface()); err != nil {
		return errors.Wrap(err, "fillDestination row value into primitive type")
	}
	return nil
}

func ensureDistinctColumns(rows pgx.Rows) error {
	seen := make(map[string]bool, len(rows.FieldDescriptions()))
	for _, columnDesc := range rows.FieldDescriptions() {
		column := string(columnDesc.Name)
		if _, ok := seen[column]; ok {
			return errors.Errorf("row contains duplicated column '%s'", column)
		}
		seen[column] = true
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
			// Field is unexported skip it.
			continue
		}

		dbTag := field.Tag.Get(dbStructTagKey)

		if dbTag == "-" {
			// Field is ignored, skip it.
			continue
		}

		// Field is embedded struct or pointer to struct.
		if field.Anonymous {
			childType := field.Type
			if field.Type.Kind() == reflect.Ptr {
				childType = field.Type.Elem()
			}
			if childType.Kind() == reflect.Struct {
				childMap, err := getColumnToFieldIndexMap(childType)
				if err != nil {
					return nil, errors.WithStack(err)
				}
				for childColumn, childIndex := range childMap {
					column := childColumn
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
