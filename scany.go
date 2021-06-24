package scany

import (
	"reflect"
	"sort"
	"strings"

	"github.com/georgysavva/scany/internal/structref"
)

// Wildcard returns an expression for populating a given Go struct
// after querying a SQL database.
//
// For example, for the following struct it should return "name", "age"
// type Pet struct {
// 	Name string
// 	Age int
// }
//
// This ensures scany keeps working if you add a field to your tables,
// making migrations easier.
// It also has the additional benefit of only requesting data that is
// used on your structs, instead of getting all columns with "SELECT *".
//
// The "db" key in the struct field's tag can specify the "json" option
// when a JSON or JSONB data type is used.
func Wildcard(v interface{}) string {
	elems := Fields(v)
	// Logic below based on strings.Join.
	if len(elems) == 0 {
		return ""
	}
	n := len(",") * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var b strings.Builder
	b.Grow(n)
	for n, s := range elems {
		if n != 0 {
			b.WriteString(`,`)
		}
		b.WriteString(`"`)
		b.WriteString(s)
		b.WriteString(`"`)
		// Alias any field containing a dot to avoid output column ambiguity,
		// as required by scany to handle nested structs.
		if strings.ContainsRune(s, '.') {
			b.WriteString(` as "`)
			b.WriteString(s)
			b.WriteString(`"`)
		}
	}
	return b.String()
}

// Fields returns column names for a SQL table that can be queried by a given Go struct.
func Fields(v interface{}) []string {
	if v == nil {
		return nil
	}
	var rv reflect.Type
	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		rv = reflect.TypeOf(v).Elem()
	} else {
		rv = reflect.Indirect(reflect.ValueOf(v)).Type()
	}
	type column struct {
		indices []int
		name    string
	}

	var cs []column
	for name, i := range structref.GetColumnToFieldIndexMap(rv) {
		cs = append(cs, column{
			indices: i,
			name:    name,
		})
	}
	// Sort output respecting structs ordering.
	sort.SliceStable(cs, func(i, j int) bool {
		a, b := cs[i].indices, cs[j].indices
		// Go inwards each nested field until the end:
		// indices a and b represent the path to the left and right fields being sorted.
		for {
			switch {
			case len(a) == 0:
				return false
			case len(b) == 0:
				return true
			case a[0] < b[0]:
				return true
			case a[0] > b[0]:
				return false
			}
			a, b = a[1:], b[1:]
		}
	})

	var columns []string
	for _, column := range cs {
		columns = append(columns, column.name)
	}
	return columns
}
