package dbscan

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type queryRowsFunc func(t *testing.T, query string) Rows

func DoTestRowScannerStartCalledExactlyOnce(t *testing.T, queryRows queryRowsFunc) {
	query := `
		SELECT *
		FROM (
			VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
		) AS t (foo, bar)
	`
	rows := queryRows(t, query)
	defer rows.Close() // nolint: errcheck

	mockStart := &mockStartScannerFunc{}
	rs := &RowScanner{rows: rows, start: mockStart.Execute}
	mockStart.On("Execute", rs, mock.AnythingOfType("reflect.Value")).Return(nil).Run(func(args mock.Arguments) {
		rs := args.Get(0).(*RowScanner)
		rs.columns = []string{"foo", "bar"}
		rs.columnToFieldIndex = map[string][]int{"foo": {0}, "bar": {1}}
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

func TestColumnToFieldIndexMap(t *testing.T) {
	type Node struct {
		ID     string
		Parent *Node
	}

	type User struct {
		ID   int
		Name string
	}

	type UserNode struct {
		User
		*Node
		CreatedBy *string
	}

	type testCase struct {
		structType   reflect.Type
		expectedCols []string
	}

	MaxStructRecursionLevel = 2
	testCases := []testCase{
		{reflect.TypeOf(Node{}), []string{"id", "parent", "parent.id", "parent.parent"}},
		{reflect.TypeOf(User{}), []string{"id", "name"}},
		{reflect.TypeOf(UserNode{}), []string{"id", "name", "created_by", "parent", "parent.id", "parent.parent"}},
	}
	for _, tc := range testCases {
		colIdxMap := getColumnToFieldIndexMap(tc.structType)
		assert.Len(t, colIdxMap, len(tc.expectedCols))
		for _, col := range tc.expectedCols {
			_, exist := colIdxMap[col]
			assert.True(t, exist)
		}
	}
}
