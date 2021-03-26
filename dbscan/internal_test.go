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

type Level1 struct {
	Level2
	Foo string
	Sub Sub
}

type Sub struct {
	Foo string
}

type Level2 struct {
	Level3
}

type Level3 struct {
	Level4
}

type Level4 struct {
	Zero  string
	One   string
	Two   string `db:"-"`
	Three string `db:"not_three"`
}

func TestFieldIndex(t *testing.T) {
	s := Level1{}
	m := getColumnToFieldIndexMap(reflect.TypeOf(s))

	assert.Equal(t, []int{0, 0, 0, 0}, m["zero"])
	assert.Equal(t, []int{0, 0, 0, 1}, m["one"])
	assert.Equal(t, []int{0, 0, 0, 3}, m["not_three"])
	assert.Equal(t, []int{1}, m["foo"])
	_, found := m["two"]
	assert.Falsef(t, found, "column two should be absent")
	assert.Equal(t, []int{2, 0}, m["sub.foo"])
}
