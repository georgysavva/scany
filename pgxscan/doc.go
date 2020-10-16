// Package pgxscan allows scanning data into Go structs and other composite types,
// when working with pgx library native interface.
/*
Essentially, pgxscan is a wrapper around github.com/georgysavva/scany/dbscan package.
pgxscan connects github.com/jackc/pgx/v4 native interface with dbscan functionality.
It contains adapters that are meant to work with pgx.Rows and proxy all calls to dbscan.
pgxscan provides all capabilities available in dbscan.
It's encouraged to read dbscan docs first to get familiar with all concepts and features:
https://pkg.go.dev/github.com/georgysavva/scany/dbscan

Querying rows

pgxscan can query rows and work with *pgxpool.Pool, *pgx.Conn or pgx.Tx directly.
To support this it has two high-level functions Select and Get,
they accept anything that implements Querier interface and query rows from it.
This means that they can be used with *pgxpool.Pool, *pgx.Conn or pgx.Tx.

Note about pgx custom types

pgx has a concept of custom types: https://pkg.go.dev/github.com/jackc/pgx/v4?tab=doc#hdr-Custom_Type_Support.

In order to use them with pgxscan you must specify your custom types by value, not by a pointer.
Let's take the pgx custom type pgtype.Text as an example:

	type User struct {
		ID   string
		Name *pgtype.Text // pgxscan won't be able to scan data into a field defined that way.
		Bio  pgtype.Text // This is a valid use of pgx custom types, pgxscan will handle it easily.
	}

This happens because struct fields are always passed to the underlying pgx.Rows.Scan() by pointer,
and if the field type is *pgtype.Text, pgx.Rows.Scan() will receive **pgtype.Text type.
pgx can't handle **pgtype.Text, since only *pgtype.Text implements pgx custom type interface.

Supported pgx version

pgxscan only works with pgx v4. So the import path of your pgx must be: "github.com/jackc/pgx/v4".
*/
package pgxscan
