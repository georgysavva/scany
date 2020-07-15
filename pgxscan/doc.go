// Package pgxscan allows scanning data into Go structs and other composite types,
// when working with pgx library.
/*
Essentially, pgxscan is a wrapper around github.com/georgysavva/scany/dbscan package.
pgxscan connects github.com/jackc/pgx/v4 with dbscan functionality.
It contains adapters that are meant to work with pgx.Rows and proxy all calls to dbscan.
pgxscan mirrors all capabilities provided by dbscan.
It's encouraged to read dbscan docs first to get familiar with all concepts and features.

How to use

The most common way to work with pgxscan is to call Select or Get functions.

Use Select to query multiple records:

	type User struct {
		ID string
		Name   string
		Email  string
		Age    int
	}

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var users []*User
	pgxscan.Select(ctx, db, &users, `SELECT id, name, email, age FROM users`)
	// users variable now contains data from all rows.

Use Get to query exactly one record:

	type User struct {
		ID string
		Name   string
		Email  string
		Age    int
	}

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var user User
	pgxscan.Get(ctx, db, &user, `SELECT id, name, email, age FROM users WHERE id='bob'`)
	// user variable now contains data from the single row.

Note about pgx custom types

pgx has a concept of custom types: https://pkg.go.dev/github.com/jackc/pgx/v4?tab=doc#hdr-Custom_Type_Support.

In order to use them with pgxscan you must specify your custom types by value, not by a pointer.
Let's take the pgx custom type pgtype.Text as an example:

	type User struct {
		ID string
		Name   *pgtype.Text // pgxscan won't be able to scan data into a field defined that way.
		Bio    pgtype.Text // This is a valid use of pgx custom types, pgxscan will handle it easily.
	}

This happens because struct fields are always passed to the underlying pgx.Rows.Scan() as addresses,
and if the field type is *pgtype.Text, pgx.Rows.Scan() will receive **pgtype.Text type.
pgx can't handle **pgtype.Text, since only *pgtype.Text implements pgx custom type interface.

Supported pgx version

pgxscan only works with pgx v4. So the import path of your pgx must be: github.com/jackc/pgx/v4
*/
package pgxscan
