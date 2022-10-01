package dbscan

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
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

// ScanAll is a package-level helper function that uses the DefaultAPI object.
// See API.ScanAll for details.
func ScanAll(dst interface{}, rows Rows) error {
	return DefaultAPI.ScanAll(dst, rows)
}

// ScanOne is a package-level helper function that uses the DefaultAPI object.
// See API.ScanOne for details.
func ScanOne(dst interface{}, rows Rows) error {
	return DefaultAPI.ScanOne(dst, rows)
}

// NameMapperFunc is a function type that maps a struct field name to the database column name.
type NameMapperFunc func(string) string

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// SnakeCaseMapper is a NameMapperFunc that maps struct field to snake case.
func SnakeCaseMapper(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// API is the core type in dbscan. It implements all the logic and exposes functionality available in the package.
// With API type users can create a custom API instance and override default settings hence configure dbscan.
type API struct {
	structTagKey          string
	columnSeparator       string
	fieldMapperFn         NameMapperFunc
	scannableTypesOption  []interface{}
	scannableTypesReflect []reflect.Type
	allowUnknownColumns   bool
}

// APIOption is a function type that changes API configuration.
type APIOption func(api *API)

// NewAPI creates a new API object with provided list of options.
func NewAPI(opts ...APIOption) (*API, error) {
	api := &API{
		structTagKey:        "db",
		columnSeparator:     ".",
		fieldMapperFn:       SnakeCaseMapper,
		allowUnknownColumns: false,
	}
	for _, o := range opts {
		o(api)
	}
	for _, stOpt := range api.scannableTypesOption {
		st := reflect.TypeOf(stOpt)
		if st == nil {
			return nil, fmt.Errorf("scany: scannable type must be a pointer, got %T", st)
		}
		if st.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("scany: scannable type must be a pointer, got %s: %s",
				st.Kind(), st.String())
		}
		st = st.Elem()
		if st.Kind() != reflect.Interface {
			return nil, fmt.Errorf("scany: scannable type must be a pointer to an interface, got %s: %s",
				st.Kind(), st.String())
		}
		api.scannableTypesReflect = append(api.scannableTypesReflect, st)
	}
	return api, nil
}

// WithStructTagKey allows to use a custom struct tag key.
// The default tag key is `db`.
func WithStructTagKey(tagKey string) APIOption {
	return func(api *API) {
		api.structTagKey = tagKey
	}
}

// WithColumnSeparator allows to use a custom separator character for column name when combining nested structs.
// The default separator is "." character.
func WithColumnSeparator(separator string) APIOption {
	return func(api *API) {
		api.columnSeparator = separator
	}
}

// WithFieldNameMapper allows to use a custom function to map field name to column names.
// The default function is SnakeCaseMapper.
func WithFieldNameMapper(mapperFn NameMapperFunc) APIOption {
	return func(api *API) {
		api.fieldMapperFn = mapperFn
	}
}

// WithScannableTypes specifies a list of interfaces that underlying database library can scan into.
// In case the destination type passed to dbscan implements one of those interfaces,
// dbscan will handle it as primitive type case i.e. simply pass the destination to the database library.
// Instead of attempting to map database columns to destination struct fields or map keys.
// In order for reflection to capture the interface type, you must pass it by pointer.
//
// For example your database library defines a scanner interface like this:
//
//	type Scanner interface {
//	    Scan(...) error
//	}
//
// You can pass it to dbscan this way:
// dbscan.WithScannableTypes((*Scanner)(nil)).
func WithScannableTypes(scannableTypes ...interface{}) APIOption {
	return func(api *API) {
		api.scannableTypesOption = scannableTypes
	}
}

// WithAllowUnknownColumns allows the scanner to ignore db columns that doesn't exist at the destination.
// The default function is to throw an error when a db column ain't found at the destination.
func WithAllowUnknownColumns(allowUnknownColumns bool) APIOption {
	return func(api *API) {
		api.allowUnknownColumns = allowUnknownColumns
	}
}

// ScanAll iterates all rows to the end. After iterating it closes the rows,
// and propagates any errors that could pop up.
// It expects that destination should be a slice. For each row it scans data and appends it to the destination slice.
// ScanAll supports both types of slices: slice of structs by a pointer and slice of structs by value,
// for example:
//
//	type User struct {
//	    ID    string
//	    Name  string
//	    Email string
//	    Age   int
//	}
//
//	var usersByPtr []*User
//	var usersByValue []User
//
// Both usersByPtr and usersByValue are valid destinations for ScanAll function.
//
// Before starting, ScanAll resets the destination slice,
// so if it's not empty it will overwrite all existing elements.
func (api *API) ScanAll(dst interface{}, rows Rows) error {
	return api.processRows(dst, rows, true /* multipleRows. */)
}

// ScanOne iterates all rows to the end and makes sure that there was exactly one row
// otherwise it returns an error. Use NotFound function to check if there were no rows.
// After iterating ScanOne closes the rows,
// and propagates any errors that could pop up.
// It scans data from that single row into the destination.
func (api *API) ScanOne(dst interface{}, rows Rows) error {
	return api.processRows(dst, rows, false /* multipleRows. */)
}

// NotFound returns true if err is a not found error.
// This error is returned by ScanOne if there were no rows.
func NotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// ErrNotFound is returned by ScanOne if there were no rows.
var ErrNotFound = errors.New("scany: no row was found")

type sliceDestinationMeta struct {
	val             reflect.Value
	elementBaseType reflect.Type
	elementByPtr    bool
}

func (api *API) processRows(dst interface{}, rows Rows, multipleRows bool) error {
	defer rows.Close() //nolint: errcheck
	var sliceMeta *sliceDestinationMeta
	if multipleRows {
		var err error
		sliceMeta, err = api.parseSliceDestination(dst)
		if err != nil {
			return fmt.Errorf("parsing slice destination: %w", err)
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
			return fmt.Errorf("scanning: %w", err)
		}
		rowsAffected++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("scany: rows final error: %w", err)
	}

	if err := rows.Close(); err != nil {
		return fmt.Errorf("scany: close rows after processing: %w", err)
	}

	exactlyOneRow := !multipleRows
	if exactlyOneRow {
		if rowsAffected == 0 {
			return ErrNotFound
		} else if rowsAffected > 1 {
			return fmt.Errorf("scany: expected 1 row, got: %d", rowsAffected)
		}
	}
	return nil
}

func (api *API) parseSliceDestination(dst interface{}) (*sliceDestinationMeta, error) {
	dstValue, err := parseDestination(dst)
	if err != nil {
		return nil, fmt.Errorf("scany: parsing destination: %w", err)
	}

	dstType := dstValue.Type()

	if dstValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf(
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
		if elementBaseTypeElem.Kind() == reflect.Struct && !api.isScannableType(elementBaseType) {
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
		return fmt.Errorf("scanning: %w", err)
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

// ScanRow is a package-level helper function that uses the DefaultAPI object.
// See API.ScanRow for details.
func ScanRow(dst interface{}, rows Rows) error {
	return DefaultAPI.ScanRow(dst, rows)
}

// ScanRow creates a new RowScanner and calls RowScanner.Scan
// that scans current row data into the destination.
// It's just a helper function if you don't bother with efficiency
// and don't want to instantiate a new RowScanner before iterating the rows,
// so it could cache the reflection work between Scan calls.
// See RowScanner for details.
func (api *API) ScanRow(dst interface{}, rows Rows) error {
	rs := api.NewRowScanner(rows)
	return rs.Scan(dst)
}

func (api *API) isScannableType(dstType reflect.Type) bool {
	dstRefType := reflect.PtrTo(dstType)
	for _, st := range api.scannableTypesReflect {
		if dstRefType.Implements(st) || dstType.Implements(st) {
			return true
		}
	}
	return false
}

func parseDestination(dst interface{}) (reflect.Value, error) {
	dstVal := reflect.ValueOf(dst)

	if !dstVal.IsValid() || (dstVal.Kind() == reflect.Ptr && dstVal.IsNil()) {
		return reflect.Value{}, fmt.Errorf("scany: destination must be a non nil pointer")
	}
	if dstVal.Kind() != reflect.Ptr {
		return reflect.Value{}, fmt.Errorf("scany: destination must be a pointer, got: %v", dstVal.Type())
	}

	dstVal = dstVal.Elem()
	return dstVal, nil
}

func mustNewAPI(opts ...APIOption) *API {
	api, err := NewAPI(opts...)
	if err != nil {
		panic(err)
	}
	return api
}

// DefaultAPI is the default instance of API with all configuration settings set to default.
var DefaultAPI = mustNewAPI()
