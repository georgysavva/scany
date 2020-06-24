// Package pgxscan allows scanning data from pgx.Rows into complex Go types.
/*
pgxscan is a wrapper around github.com/georgysavva/dbscan package.
It contains adapters and proxy functions that are meant to connect github.com/jackc/pgx/v4
with github.com/georgysavva/dbscan functionality. pgxscan mirrors all capabilities provided by dbscan.
See dbscan docs to get familiar with all details and features.

How to use

The most common way to use pgxscan is by calling QueryAll or QueryOne function,
it's as simple as this:

	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	// Use QueryAll to query multiple records.
	var users []*User
	if err := pgxscan.QueryAll(
		ctx, &users, db, `SELECT user_id, name, email, age from users`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// users variable now contains data from all rows.

	// Use QueryOne to query exactly one record.
	var user User
	if err := pgxscan.QueryOne(
		ctx, &user, db, `SELECT user_id, name, email, age from users where id='bob'`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// users variable now contains data from all rows.
*/
package pgxscan
