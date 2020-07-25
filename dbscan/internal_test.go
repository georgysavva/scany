package dbscan

import (
	"reflect"
)

func NewMockStartScannerFunc() *mockStartScannerFunc { return &mockStartScannerFunc{} }

func NewRowScannerWithStart(rows Rows, start startScannerFunc) *RowScanner {
	return &RowScanner{rows: rows, start: start}
}

func PatchRowScanner(rs *RowScanner, columns []string, columnToFieldIndex map[string][]int, mapElementType reflect.Type) {
	rs.columns = columns
	rs.columnToFieldIndex = columnToFieldIndex
	rs.mapElementType = mapElementType
}
