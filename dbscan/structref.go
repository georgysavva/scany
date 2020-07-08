package dbscan

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var dbStructTagKey = "db"

func getColumnToFieldIndexMap(structType reflect.Type) (map[string][]int, error) {
	result := make(map[string][]int, structType.NumField())

	setColumn := func(column string, index []int) error {
		if otherIndex, ok := result[column]; ok {
			return errors.Errorf(
				"scany: Column must have exactly one field pointing to it; "+
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
					// the default behavior is to propagate columns as is.
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

var matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCapRe = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
