package testutil

import (
	"database/sql"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pkg/errors"
)

func StartCrdbServer() (*testserver.TestServer, error) {
	ts, err := testserver.NewTestServer()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := ts.Start(); err != nil {
		return nil, errors.WithStack(err)
	}

	url := ts.PGURL()
	if url == nil {
		return nil, errors.New("test server doesn't have the URL")
	}

	sqlDB, err := sql.Open("pgx", url.String())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer sqlDB.Close()
	if err := ts.WaitForInit(sqlDB); err != nil {
		return nil, errors.WithStack(err)
	}
	return ts, nil
}
