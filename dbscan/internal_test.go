package dbscan

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func DoTestRowScannerStartCalledExactlyOnce(
	t *testing.T,
	rows Rows, columns []string,
	columnToFieldIndex map[string][]int, mapElementType reflect.Type,
) {
	mockStart := &mockStartScannerFunc{}
	rs := &RowScanner{rows: rows, start: mockStart.Execute}
	mockStart.On("Execute", rs, mock.AnythingOfType("reflect.Value")).Return(nil).Run(func(args mock.Arguments) {
		rs := args.Get(0).(*RowScanner)
		rs.columns = columns
		rs.columnToFieldIndex = columnToFieldIndex
		rs.mapElementType = mapElementType
	})
	for rows.Next() {
		dst := &struct {
			Foo string
		}{}
		err := rs.Scan(dst)
		require.NoError(t, err)
	}

	mockStart.AssertNumberOfCalls(t, "Execute", 1)
}
