package sqlscan

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/georgysavva/scany/dbscan"
)

// Querier is something that sqlscan can query and get the *sql.Rows from.
// For example, it can be: *sql.DB, *sql.Conn or *sql.Tx.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

var (
	_ Querier = &sql.DB{}
	_ Querier = &sql.Conn{}
	_ Querier = &sql.Tx{}
)

// Select is a package-level helper function that uses the DefaultAPI object.
// See API.Select for details.
func Select(ctx context.Context, db Querier, dst interface{}, query string, args ...interface{}) error {
	return errors.WithStack(DefaultAPI.Select(ctx, db, dst, query, args...))
}

// Get is a package-level helper function that uses the DefaultAPI object.
// See API.Get for details.
func Get(ctx context.Context, db Querier, dst interface{}, query string, args ...interface{}) error {
	return errors.WithStack(DefaultAPI.Get(ctx, db, dst, query, args...))
}

// ScanAll is a package-level helper function that uses the DefaultAPI object.
// See API.ScanAll for details.
func ScanAll(dst interface{}, rows *sql.Rows) error {
	return errors.WithStack(DefaultAPI.ScanAll(dst, rows))
}

// ScanOne is a package-level helper function that uses the DefaultAPI object.
// See API.ScanOne for details.
func ScanOne(dst interface{}, rows *sql.Rows) error {
	return errors.WithStack(DefaultAPI.ScanOne(dst, rows))
}

// RowScanner is a wrapper around the dbscan.RowScanner type.
// See dbscan.RowScanner for details.
type RowScanner struct {
	*dbscan.RowScanner
}

// NewRowScanner is a package-level helper function that uses the DefaultAPI object.
// See API.NewRowScanner for details.
func NewRowScanner(rows *sql.Rows) *RowScanner {
	return DefaultAPI.NewRowScanner(rows)
}

// ScanRow is a package-level helper function that uses the DefaultAPI object.
// See API.ScanRow for details.
func ScanRow(dst interface{}, rows *sql.Rows) error {
	return DefaultAPI.ScanRow(dst, rows)
}

// API is a wrapper around the dbscan.API type.
// See dbscan.API for details.
type API struct {
	dbscanAPI *dbscan.API
}

// NewAPI creates new API instance from dbscan.API instance.
func NewAPI(dbscanAPI *dbscan.API) *API {
	api := &API{dbscanAPI: dbscanAPI}
	return api
}

// Select is a high-level function that queries rows from Querier and calls the ScanAll function.
// See ScanAll for details.
func (api *API) Select(ctx context.Context, db Querier, dst interface{}, query string, args ...interface{}) error {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "scany: query multiple result rows")
	}
	err = api.ScanAll(dst, rows)
	return errors.WithStack(err)
}

// Get is a high-level function that queries rows from Querier and calls the ScanOne function.
// See ScanOne for details.
func (api *API) Get(ctx context.Context, db Querier, dst interface{}, query string, args ...interface{}) error {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "scany: query one result row")
	}
	err = api.ScanOne(dst, rows)
	return errors.WithStack(err)
}

// ScanAll is a wrapper around the dbscan.ScanAll function.
// See dbscan.ScanAll for details.
func (api *API) ScanAll(dst interface{}, rows *sql.Rows) error {
	err := api.dbscanAPI.ScanAll(dst, rows)
	return errors.WithStack(err)
}

// ScanOne is a wrapper around the dbscan.ScanOne function.
// See dbscan.ScanOne for details. If no rows are found it
// returns an sql.ErrNoRows error.
func (api *API) ScanOne(dst interface{}, rows *sql.Rows) error {
	err := api.dbscanAPI.ScanOne(dst, rows)
	if dbscan.NotFound(err) {
		return errors.WithStack(sql.ErrNoRows)
	}
	return errors.WithStack(err)
}

// NotFound is a helper function to check if an error
// is `sql.ErrNoRows`.
func NotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// NewRowScanner returns a new RowScanner instance.
func (api *API) NewRowScanner(rows *sql.Rows) *RowScanner {
	return &RowScanner{RowScanner: api.dbscanAPI.NewRowScanner(rows)}
}

// ScanRow is a wrapper around the dbscan.ScanRow function.
// See dbscan.ScanRow for details.
func (api *API) ScanRow(dst interface{}, rows *sql.Rows) error {
	err := api.dbscanAPI.ScanRow(dst, rows)
	return errors.WithStack(err)
}

// DefaultAPI is the default instance of API that is wrapped around the dbscan.DefaultAPI instance.
var DefaultAPI = NewAPI(dbscan.DefaultAPI)
