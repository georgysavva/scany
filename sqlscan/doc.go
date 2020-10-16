// Package sqlscan allows scanning data into Go structs and other composite types,
// when working with database/sql library.
/*
Essentially, sqlscan is a wrapper around github.com/georgysavva/scany/dbscan package.
sqlscan connects database/sql with dbscan functionality.
It contains adapters that are meant to work with *sql.Rows and proxy all calls to dbscan.
sqlscan provides all capabilities available in dbscan.
It's encouraged to read dbscan docs first to get familiar with all concepts and features:
https://pkg.go.dev/github.com/georgysavva/scany/dbscan

Querying rows

sqlscan can query rows and work with *sql.DB, *sql.Conn or *sql.Tx directly.
To support this it has two high-level functions Select and Get,
they accept anything that implements Querier interface and query rows from it.
This means that they can be used with *sql.DB, *sql.Conn or *sql.Tx.
*/
package sqlscan
