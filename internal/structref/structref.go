package structref

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

// GetColumnToFieldIndexMap containing where columns should be mapped.
func GetColumnToFieldIndexMap(structType reflect.Type) map[string][]int {
	result := make(map[string][]int, structType.NumField())
	jsonColumns := map[string]struct{}{}
	var queue []*toTraverse
	queue = append(queue, &toTraverse{Type: structType, IndexPrefix: nil, ColumnPrefix: ""})
	for len(queue) > 0 {
		traversal := queue[0]
		queue = queue[1:]
		structType := traversal.Type
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)

			if field.PkgPath != "" && !field.Anonymous {
				// Field is unexported, skip it.
				continue
			}

			dbTag, dbTagPresent := field.Tag.Lookup(dbStructTagKey)
			var options tagOptions
			if dbTagPresent {
				dbTag, options = parseTag(dbTag)
			}
			if dbTag == "-" {
				// Field is ignored, skip it.
				continue
			}

			index := make([]int, 0, len(traversal.IndexPrefix)+len(field.Index))
			index = append(index, traversal.IndexPrefix...)
			index = append(index, field.Index...)

			columnPart := dbTag
			if !dbTagPresent {
				columnPart = toSnakeCase(field.Name)
			}

			childType := field.Type
			if field.Type.Kind() == reflect.Ptr {
				childType = field.Type.Elem()
			}
			if childType.Kind() == reflect.Struct {
				if field.Anonymous {
					// If "db" tag is present for embedded struct
					// use it with "." to prefix all column from the embedded struct.
					// the default behavior is to propagate columns as is.
					columnPart = dbTag
				}
			}

			column := buildColumn(traversal.ColumnPrefix, columnPart)
			if childType.Kind() == reflect.Struct {
				if options.Contains("json") {
					jsonColumns[column] = struct{}{}
				} else {
					queue = append(queue, &toTraverse{
						Type:         childType,
						IndexPrefix:  index,
						ColumnPrefix: column,
					})
				}
			}
			if !field.Anonymous {
				_, self := jsonColumns[column]
				_, parent := jsonColumns[traversal.ColumnPrefix]
				if !self || !parent {
					if _, exists := result[column]; !exists {
						result[column] = index
					}
				}
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

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
