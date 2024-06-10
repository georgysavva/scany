package dbscan_test

import (
	"testing"

	"github.com/georgysavva/scany/v2/dbscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanAllType(t *testing.T) {
	type stype struct {
		Foo string
		Bar string
	}

	query := `
				SELECT *
				FROM (
					VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
				) AS t (foo, bar)
			`

	expected := []stype{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}

	rows := queryRows(t, query)
	dst, err := dbscan.APIScanAllType[stype](testAPI, rows)
	require.NoError(t, err)
	require.Equal(t, expected, dst)
}

func TestScanOneType(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, singleRowsQuery)
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	got, err := dbscan.APIScanOneType[testModel](testAPI, rows)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestScanRowType(t *testing.T) {
	t.Parallel()
	rows := queryRows(t, singleRowsQuery)
	defer rows.Close() // nolint: errcheck
	rows.Next()
	expected := testModel{Foo: "foo val", Bar: "bar val"}

	var got testModel
	got, err := dbscan.APIScanRowType[testModel](testAPI, rows)
	require.NoError(t, err)
	requireNoRowsErrorsAndClose(t, rows)

	assert.Equal(t, expected, got)
}
