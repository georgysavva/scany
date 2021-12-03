package dbscan

import (
	"reflect"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type toTraverse struct {
	Type         reflect.Type
	IndexPrefix  []int
	ColumnPrefix string
}

type structref struct {
	fieldIndexes map[string][]int
}

//fieldByName tries to find the field from the struct with the name
func (sr structref) fieldByName(e reflect.Value, name string) reflect.Value {
	return e.FieldByIndex(sr.fieldIndexes[name])
}

//fieldValue is used for getting the fields value
func (sr structref) fieldValue(e reflect.Value, name string) (interface{}, error) {
	value := sr.fieldByName(e, name)

	if value.IsValid() {
		return value.Interface(), nil
	}

	return "", errors.New("field '" + name + "' not found")
}

func (api *API) getColumnToFieldIndexMapV2(structType reflect.Type) structref {
	return structref{api.getColumnToFieldIndexMap(structType)}
}

var fieldIndexMapCache sync.Map

func (api *API) getColumnToFieldIndexMap(structType reflect.Type) map[string][]int {
	fieldIndexMap, found := fieldIndexMapCache.Load(structType)
	if found {
		return fieldIndexMap.(map[string][]int)
	}

	//When field index map is not found from the cache it computes it and stores it into the cache
	newFieldIndexMap := api.makeColumnToFieldIndexMap(structType)
	fieldIndexMapCache.Store(structType, newFieldIndexMap)
	return newFieldIndexMap
}

//makeColumnToFieldIndexMap is a function that generates the map that is used to determine which database table column is mapped to which struct field
func (api *API) makeColumnToFieldIndexMap(structType reflect.Type) map[string][]int {
	result := make(map[string][]int, structType.NumField())
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

			dbTag, dbTagPresent := field.Tag.Lookup(api.structTagKey)
			if dbTagPresent {
				dbTag = strings.Split(dbTag, ",")[0]
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
				columnPart = api.fieldMapperFn(field.Name)
			}
			if !field.Anonymous {
				column := api.buildColumn(traversal.ColumnPrefix, columnPart)

				if _, exists := result[column]; !exists {
					result[column] = index
				}
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
				columnPrefix := api.buildColumn(traversal.ColumnPrefix, columnPart)
				queue = append(queue, &toTraverse{
					Type:         childType,
					IndexPrefix:  index,
					ColumnPrefix: columnPrefix,
				})
			}
		}
	}

	return result
}

func (api *API) buildColumn(parts ...string) string {
	var notEmptyParts []string
	for _, p := range parts {
		if p != "" {
			notEmptyParts = append(notEmptyParts, p)
		}
	}
	return strings.Join(notEmptyParts, api.columnSeparator)
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
