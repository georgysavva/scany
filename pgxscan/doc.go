// Package pgxscan allows scanning data into Go structs and other composite types,
// when working with pgx library.
/*
Essentially, pgxscan is a wrapper around github.com/georgysavva/scany/dbscan package.
pgxscan connects github.com/jackc/pgx/v4 with dbscan functionality.
It contains adapters that are meant to work with pgx.Rows and proxy all calls to dbscan.
pgxscan mirrors all capabilities provided by dbscan.
It's encouraged to read dbscan docs first to get familiar with all concepts and features.

How to use

The most common way to use pgxscan is by calling Query or QueryOne function,
it's as simple as this:

	type User struct {
		UserID string
		Name   string
		Email  string
		Age    int
	}

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	// Use Query to query multiple records.
	var users []*User
	pgxscan.Query(ctx, &users, db, `SELECT user_id, name, email, age FROM users`)
	// users variable now contains data from all rows.

Pgx custom types

pgx has a concept of custom types: https://pkg.go.dev/github.com/jackc/pgx/v4?tab=doc#hdr-Custom_Type_Support.

You can use them with pgxscan too, here is an example of a struct with pgtype.Text field:

	type User struct {
		UserID string
		Name   string
		Bio    pgtype.Text
	}

Note that you must specify pgtype.Text by value, not by a pointer. This will not work:

	type User struct {
		UserID string
		Name   string
		Bio    *pgtype.Text // pgxscan won't be able to scan data into a field defined that way.
	}

This happens because struct fields are always passed to the underlying pgx.Rows.Scan() as pointers,
and if the field type is *pgtype.Text, pgx.Rows.Scan() will receive **pgtype.Text and
pgx won't be able to handle that type, since only *pgtype.Text implements pgx custom type interface.

Supported pgx version

pgxscan only works with pgx v4. So the import path of your pgx must be: github.com/jackc/pgx/v4
*/
package pgxscan
