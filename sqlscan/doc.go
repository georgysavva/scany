// Package sqlscan allows scanning data into Go structs and other composite types,
// when working with database/sql library.
/*
Essentially, sqlscan is a wrapper around github.com/georgysavva/scany/dbscan package.
sqlscan connects database/sql with dbscan functionality.
It contains adapters that are meant to work with *sql.Rows and proxy all calls to dbscan.
sqlscan mirrors all capabilities provided by dbscan.
It's encouraged to read dbscan docs first to get familiar with all concepts and features.

How to use

The most common way to work with sqlscan is to call Select or Get functions.

Use Select to query multiple records:

	type User struct {
		ID string
		Name   string
		Email  string
		Age    int
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	var users []*User
	sqlscan.Select(ctx, db, &users, `SELECT id, name, email, age FROM users`)
	// users variable now contains data from all rows.

Use Get to query exactly one record:

	type User struct {
		ID string
		Name   string
		Email  string
		Age    int
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	var user User
	sqlscan.Get(ctx, db, &user, `SELECT id, name, email, age FROM users WHERE id='bob'`)
	// user variable now contains data from the single row.
*/
package sqlscan
