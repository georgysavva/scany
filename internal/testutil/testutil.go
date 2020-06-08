package testutil

import (
	"database/sql"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func StartCrdbServer() *testserver.TestServer {
	ts, err := testserver.NewTestServer()
	if err != nil {
		panic(err)
	}
	if err := ts.Start(); err != nil {
		panic(err)
	}

	url := ts.PGURL()
	if url == nil {
		panic("test server doesn't have the url")
	}

	sqlDB, err := sql.Open("pgx", url.String())
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()
	if err := ts.WaitForInit(sqlDB); err != nil {
		panic(err)
	}
	return ts

}
