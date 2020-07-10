// Package sqlscan allows scanning data into Go structs and other composite types,
// when working with database/sql library.
/*
Essentially, sqlscan is a wrapper around github.com/georgysavva/scany/dbscan package.
sqlscan connects database/sql with dbscan functionality.
It contains adapters that are meant to work with *sql.Rows and proxy all calls to dbscan.
sqlscan mirrors all capabilities provided by dbscan.
It's encouraged to read dbscan docs first to get familiar with all concepts and features.

How to use

The most common way to use sqlscan is by calling Query or QueryOne function,
it's as simple as this:

	type User struct {
		UserID string
		Name   string
		Email  string
		Age    int
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	// Use Query to query multiple records.
	var users []*User
	sqlscan.Query(ctx, &users, db, `SELECT user_id, name, email, age FROM users`)
	// users variable now contains data from all rows.
*/
package sqlscan
