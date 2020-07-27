package dbscan

import (
	"reflect"
	"regexp"
	"strings"
)

var dbStructTagKey = "db"

type toTraverse struct {
	Type         reflect.Type
	IndexPrefix  []int
	ColumnPrefix string
}

func getColumnToFieldIndexMap(structType reflect.Type) map[string][]int {
	result := make(map[string][]int, structType.NumField())
	var queue []*toTraverse
	queue = append(queue, &toTraverse{Type: structType, IndexPrefix: nil, ColumnPrefix: ""})
	for len(queue) > 0 {
		traversal := queue[0]
		queue = queue[1:]
		structType := traversal.Type
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
			index := append(traversal.IndexPrefix, field.Index...)
			if field.Anonymous {
				childType := field.Type
				if field.Type.Kind() == reflect.Ptr {
					childType = field.Type.Elem()
				}
				if childType.Kind() == reflect.Struct {
					// Field is embedded struct or pointer to struct.

					// If "db" tag is present for embedded struct
					// use it with "." to prefix all column from the embedded struct.
					// the default behavior is to propagate columns as is.
					columnPrefix := buildColumn(traversal.ColumnPrefix, dbTag)
					queue = append(queue, &toTraverse{
						Type:         childType,
						IndexPrefix:  index,
						ColumnPrefix: columnPrefix,
					})
					continue
				}
			}

			column := dbTag
			if dbTag == "" {
				column = toSnakeCase(field.Name)
			}
			finalColumn := buildColumn(traversal.ColumnPrefix, column)

			if _, exists := result[finalColumn]; !exists {
				result[finalColumn] = index
			}
		}
	}

	return result
}

func buildColumn(parts ...string) string {
	var notEmptyParts []string
	for _, p := range parts {
		if p != "" {
			notEmptyParts = append(notEmptyParts, p)
		}
	}
	return strings.Join(notEmptyParts, ".")
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

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
