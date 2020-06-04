package pgxscan_test

import (
	"context"
	"flag"
	"github.com/georgysavva/sqlscan/pgxscan"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/require"

	"github.com/georgysavva/sqlscan/internal/testutil"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var testDB *pgxpool.Pool

func TestRowsWrapScan_AllDestinationsAreUnknown_Succeeds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		d1   interface{}
		d2   interface{}
		d3   interface{}
	}{
		{
			name: "all destinations are *interface{}",
			d1:   new(interface{}),
			d2:   new(interface{}),
			d3:   new(interface{}),
		},
		{
			name: "none of destinations are *interface{}",
			d1:   new(string),
			d2:   new(string),
			d3:   new(string),
		},
		{
			name: "mix of *interface{} and non *interface{} destinations",
			d1:   new(interface{}),
			d2:   new(string),
			d3:   new(interface{}),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rows, clean := selectRows(t)
			defer clean()
			wr := pgxscan.WrapRows(rows)
			err := wr.Scan(tc.d1, tc.d2, tc.d3)
			require.NoError(t, err)
			assert.Equal(t, "1", reflect.ValueOf(tc.d1).Elem().Interface())
			assert.Equal(t, "2", reflect.ValueOf(tc.d2).Elem().Interface())
			assert.Equal(t, "3", reflect.ValueOf(tc.d3).Elem().Interface())
		})
	}
}

func selectRows(t *testing.T) (pgx.Rows, func()) {
	t.Helper()
	rows, err := testDB.Query(context.Background(), `select '1', '2', '3'`)
	require.NoError(t, err)
	rows.Next()
	return rows, func() {
		require.NoError(t, rows.Err())
		rows.Close()
	}
}

func TestMain(m *testing.M) {
	exitCode := func() int {
		flag.Parse()
		ts := testutil.StartCrdbServer()
		defer ts.Stop()
		var err error
		testDB, err = pgxpool.Connect(context.Background(), ts.PGURL().String())
		if err != nil {
			panic(err)
		}
		defer testDB.Close()
		return m.Run()
	}()
	os.Exit(exitCode)
}
