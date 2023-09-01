//go:build with_mssql
// +build with_mssql

package sqlscan_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/georgysavva/scany/v2/sqlscan"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	multipleSetsQueryMssql = `
	SELECT *
	FROM (
		VALUES ('foo val', 'bar val'), ('foo val 2', 'bar val 2'), ('foo val 3', 'bar val 3')
	) as t1 (foo, bar);
	
	
	SELECT *
		FROM (
			VALUES ('egg val', 'bacon val')
		) as t2 (egg, bacon);
	`
)

func getEnv(key string, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
func TestMSScanAllSets(t *testing.T) {
	t.Parallel()
	testSqliteDB, err := sql.Open("sqlserver", getEnv("MSSQL_URL", "sqlserver://sa:sa@localhost"))
	if err != nil {
		panic(err)
	}
	dbscanAPI, err := sqlscan.NewDBScanAPI()
	if err != nil {
		panic(fmt.Errorf("new DB scan API: %w", err))
	}
	api, err := sqlscan.NewAPI(dbscanAPI)
	if err != nil {
		panic(fmt.Errorf("new API: %w", err))
	}
	type testModel2 struct {
		Egg   string
		Bacon string
	}

	expected1 := []*testModel{
		{Foo: "foo val", Bar: "bar val"},
		{Foo: "foo val 2", Bar: "bar val 2"},
		{Foo: "foo val 3", Bar: "bar val 3"},
	}
	expected2 := []*testModel2{
		{Egg: "egg val", Bacon: "bacon val"},
	}

	rows, err := testSqliteDB.Query(multipleSetsQueryMssql)
	require.NoError(t, err)

	var got1 []*testModel
	var got2 []*testModel2
	err = api.ScanAllSets([]any{&got1, &got2}, rows)
	require.NoError(t, err)

	assert.Equal(t, expected1, got1)
	assert.Equal(t, expected2, got2)
}
