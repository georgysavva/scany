package scan

import (
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"reflect"
	"regexp"
	"strings"
)

type Destination struct {
	dstValue           reflect.Value
	columnToFieldIndex map[string][]int
	sliceElementType   reflect.Type
	sliceElementByPtr  bool
	mapElementType     reflect.Type
}

func ParseDst(dst interface{}, exactlyOneRow bool) (*Destination, error) {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr {
		return nil, errors.Errorf("destination must be a pointer, got: %v", dstVal.Type())
	}
	dstElemVal := dstVal.Elem()
	if !dstElemVal.IsValid() || !dstElemVal.CanSet() {
		return nil, errors.Errorf("destination must be a valid non nil pointer")
	}

	if !exactlyOneRow {
		if dstElemVal.Kind() != reflect.Slice {
			return nil, errors.Errorf(
				"destination must be a pointer to a slice, got: %v", dstVal.Type(),
			)
		}

		// Make sure that slice is empty.
		dstElemVal.Set(dstElemVal.Slice(0, 0))
	}

	return NewDestination(dstElemVal), nil
}

func NewDestination(dstValue reflect.Value) *Destination {
	return &Destination{dstValue: dstValue}
}

type RowsScanner interface {
	Scan(dst ...interface{}) error
	Values() ([]interface{}, error)
	Columns() []string
}

type RowsWrapper struct {
	pgx.Rows
}

func (rw *RowsWrapper) Columns() []string {
	columns := make([]string, len(rw.FieldDescriptions()))
	for i, field := range rw.FieldDescriptions() {
		columns[i] = string(field.Name)
	}
	return columns
}

func (d *Destination) Fill(rows RowsScanner) error {
	var err error
	if d.dstValue.Kind() == reflect.Slice {
		err = d.fillSlice(rows)
	} else {
		err = d.fillElement(d.dstValue, rows)
	}
	return errors.WithStack(err)
}

var columnStructTagKey = "db"

func GetColumnToFieldIndexMap(structType reflect.Type) (map[string][]int, error) {
	result := make(map[string][]int, structType.NumField())
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Field is unexported skip it.
		if field.PkgPath != "" {
			continue
		}

		columnName := field.Tag.Get(columnStructTagKey)
		if columnName == "" {
			columnName = toSnakeCase(field.Name)
		}
		if otherIndex, ok := result[columnName]; ok {
			return nil, errors.Errorf(
				"Column must have exactly one field pointing to it; "+
					"found 2 fields with indexes %d and %d pointing to '%s' in %v",
				otherIndex, field.Index, columnName, structType,
			)
		}
		result[columnName] = field.Index
	}
	return result, nil
}

func (d *Destination) fillSlice(rows RowsScanner) error {
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

func (d *Destination) fillElement(elementValue reflect.Value, rows RowsScanner) error {
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

func (d *Destination) fillStruct(elementValue reflect.Value, rows RowsScanner) error {
	if d.columnToFieldIndex == nil {
		var err error
		d.columnToFieldIndex, err = GetColumnToFieldIndexMap(elementValue.Type())
		if err != nil {
			return errors.WithStack(err)
		}
	}

	scans := make([]interface{}, len(rows.Columns()))
	for i, columnName := range rows.Columns() {
		fieldIndex, ok := d.columnToFieldIndex[columnName]
		if !ok {
			return errors.Errorf(
				"column: '%s': no corresponding field found or it's unexported in %v",
				columnName, elementValue.Type(),
			)
		}
		fieldVal := elementValue.FieldByIndex(fieldIndex)
		if !fieldVal.IsValid() || !fieldVal.CanSet() || !fieldVal.Addr().CanInterface() {
			return errors.Errorf(
				"column: '%s': corresponding field with index %d is invalid or can't be set in %v",
				columnName, fieldIndex, elementValue.Type(),
			)
		}
		scans[i] = fieldVal.Addr().Interface()
	}
	if err := rows.Scan(scans...); err != nil {
		return errors.Wrap(err, "scan row into struct fields")
	}
	return nil
}

func (d *Destination) fillMap(elementValue reflect.Value, rows RowsScanner) error {
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

	for i, column := range rows.Columns() {
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

func fillPrimitive(elementValue reflect.Value, rows RowsScanner) error {
	if len(rows.Columns()) != 1 {
		return errors.Errorf(
			"to scan into a primitive type, columns number must be exactly 1, got: %d",
			len(rows.Columns()),
		)
	}
	if err := rows.Scan(elementValue.Addr().Interface()); err != nil {
		return errors.Wrap(err, "scan row value into primitive type")
	}
	return nil
}

var matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCapRe = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
