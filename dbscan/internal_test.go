package dbscan

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type queryRowsFunc func(t *testing.T, query string) Rows

func DoTestRowScannerStartCalledExactlyOnce(t *testing.T, api *API, queryRows queryRowsFunc) {
	query := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	rows := queryRows(t, query)
	defer rows.Close() //nolint: errcheck

	mockStart := &mockStartScannerFunc{}
	rs := api.NewRowScanner(rows)
	rs.start = mockStart.Execute
	mockStart.On("Execute", rs, mock.AnythingOfType("reflect.Value")).Return(nil).Run(func(args mock.Arguments) {
		rs := args.Get(0).(*RowScanner)
		rs.columns = []string{"foo", "bar"}
		rs.columnToFieldIndex = map[string][]int{"foo": {0}, "bar": {1}}
		rs.scanFn = rs.scanStruct
	})

	for rows.Next() {
		dst := &struct {
			Foo string
			Bar string
		}{}
		err := rs.Scan(dst)
		require.NoError(t, err)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())

	mockStart.AssertNumberOfCalls(t, "Execute", 1)
}
