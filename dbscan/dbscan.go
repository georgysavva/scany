package dbscan

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Rows is an abstract database rows that dbscan can iterate over and get the data from.
// This interface is used to decouple from any particular database library.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Columns() ([]string, error)
	Scan(dest ...interface{}) error
}

func ScanAll(dst interface{}, rows Rows) error {
	return errors.WithStack(DefaultAPI.ScanAll(dst, rows))
}

func ScanOne(dst interface{}, rows Rows) error {
	return errors.WithStack(DefaultAPI.ScanOne(dst, rows))
}

type NameMapperFunc func(string) string

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func SnakeCaseMapper(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

type API struct {
	structTagKey    string
	columnSeparator string
	fieldMapperFn   NameMapperFunc
}

type APIOption func(api *API)

func NewAPI(opts ...APIOption) *API {
	api := &API{
		structTagKey:    "db",
		columnSeparator: ".",
		fieldMapperFn:   SnakeCaseMapper,
	}
	for _, o := range opts {
		o(api)
	}
	return api
}

func WithStructTagKey(tagKey string) APIOption {
	return func(api *API) {
		api.structTagKey = tagKey
	}
}

func WithColumnSeparator(separator string) APIOption {
	return func(api *API) {
		api.columnSeparator = separator
	}
}

func WithFieldNameMapper(mapperFn NameMapperFunc) APIOption {
	return func(api *API) {
		api.fieldMapperFn = mapperFn
	}
}

// ScanAll iterates all rows to the end. After iterating it closes the rows,
// and propagates any errors that could pop up.
// It expects that destination should be a slice. For each row it scans data and appends it to the destination slice.
// ScanAll supports both types of slices: slice of structs by a pointer and slice of structs by value,
// for example:
//
//     type User struct {
//         ID    string
//         Name  string
//         Email string
//         Age   int
//     }
//
//     var usersByPtr []*User
//     var usersByValue []User
//
// Both usersByPtr and usersByValue are valid destinations for ScanAll function.
//
// Before starting, ScanAll resets the destination slice,
// so if it's not empty it will overwrite all existing elements.
func (api *API) ScanAll(dst interface{}, rows Rows) error {
	err := api.processRows(dst, rows, true /* multipleRows */)
	return errors.WithStack(err)
}

// ScanOne iterates all rows to the end and makes sure that there was exactly one row
// otherwise it returns an error. Use NotFound function to check if there were no rows.
// After iterating ScanOne closes the rows,
// and propagates any errors that could pop up.
// It scans data from that single row into the destination.
func (api *API) ScanOne(dst interface{}, rows Rows) error {
	err := api.processRows(dst, rows, false /* multipleRows */)
	return errors.WithStack(err)
}

// NotFound returns true if err is a not found error.
// This error is returned by ScanOne if there were no rows.
func NotFound(err error) bool {
	return errors.Is(err, errNotFound)
}

var errNotFound = errors.New("scany: no row was found")

type sliceDestinationMeta struct {
	val             reflect.Value
	elementBaseType reflect.Type
	elementByPtr    bool
}

func (api *API) processRows(dst interface{}, rows Rows, multipleRows bool) error {
	defer rows.Close() // nolint: errcheck
	var sliceMeta *sliceDestinationMeta
	if multipleRows {
		var err error
		sliceMeta, err = parseSliceDestination(dst)
		if err != nil {
			return errors.WithStack(err)
		}
		// Make sure slice is empty.
		sliceMeta.val.Set(sliceMeta.val.Slice(0, 0))
	}
	rs := api.NewRowScanner(rows)
	var rowsAffected int
	for rows.Next() {
		var err error
		if multipleRows {
			err = scanSliceElement(rs, sliceMeta)
		} else {
			err = rs.Scan(dst)
		}
		if err != nil {
			return errors.WithStack(err)
		}
		rowsAffected++
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "scany: rows final error")
	}

	if err := rows.Close(); err != nil {
		return errors.Wrap(err, "scany: close rows after processing")
	}

	exactlyOneRow := !multipleRows
	if exactlyOneRow {
		if rowsAffected == 0 {
			return errors.WithStack(errNotFound)
		} else if rowsAffected > 1 {
			return errors.Errorf("scany: expected 1 row, got: %d", rowsAffected)
		}
	}
	return nil
}

func parseSliceDestination(dst interface{}) (*sliceDestinationMeta, error) {
	dstValue, err := parseDestination(dst)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	dstType := dstValue.Type()

	if dstValue.Kind() != reflect.Slice {
		return nil, errors.Errorf(
			"scany: destination must be a slice, got: %v", dstType,
		)
	}

	elementBaseType := dstType.Elem()
	var elementByPtr bool
	// If it's a slice of pointers to structs,
	// we handle it the same way as it would be slice of struct by value
	// and dereference pointers to values,
	// because eventually we work with fields.
	// But if it's a slice of primitive type e.g. or []string or []*string,
	// we must leave and pass elements as is to Rows.Scan().
	if elementBaseType.Kind() == reflect.Ptr {
		elementBaseTypeElem := elementBaseType.Elem()
		if elementBaseTypeElem.Kind() == reflect.Struct {
			elementBaseType = elementBaseTypeElem
			elementByPtr = true
		}
	}

	meta := &sliceDestinationMeta{
		val:             dstValue,
		elementBaseType: elementBaseType,
		elementByPtr:    elementByPtr,
	}
	return meta, nil
}

func scanSliceElement(rs *RowScanner, sliceMeta *sliceDestinationMeta) error {
	dstValPtr := reflect.New(sliceMeta.elementBaseType)
	if err := rs.Scan(dstValPtr.Interface()); err != nil {
		return errors.WithStack(err)
	}
	var elemVal reflect.Value
	if sliceMeta.elementByPtr {
		elemVal = dstValPtr
	} else {
		elemVal = dstValPtr.Elem()
	}

	sliceMeta.val.Set(reflect.Append(sliceMeta.val, elemVal))
	return nil
}

func ScanRow(dst interface{}, rows Rows) error {
	return errors.WithStack(DefaultAPI.ScanRow(dst, rows))
}

// ScanRow creates a new RowScanner and calls RowScanner.Scan
// that scans current row data into the destination.
// It's just a helper function if you don't bother with efficiency
// and don't want to instantiate a new RowScanner before iterating the rows,
// so it could cache the reflection work between Scan calls.
// See RowScanner for details.
func (api *API) ScanRow(dst interface{}, rows Rows) error {
	rs := api.NewRowScanner(rows)
	err := rs.Scan(dst)
	return errors.WithStack(err)
}

func parseDestination(dst interface{}) (reflect.Value, error) {
	dstVal := reflect.ValueOf(dst)

	if !dstVal.IsValid() || (dstVal.Kind() == reflect.Ptr && dstVal.IsNil()) {
		return reflect.Value{}, errors.Errorf("scany: destination must be a non nil pointer")
	}
	if dstVal.Kind() != reflect.Ptr {
		return reflect.Value{}, errors.Errorf("scany: destination must be a pointer, got: %v", dstVal.Type())
	}

	dstVal = dstVal.Elem()
	return dstVal, nil
}

var DefaultAPI = NewAPI()
