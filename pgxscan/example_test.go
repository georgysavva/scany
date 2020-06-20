package pgxscan_test

import (
	"github.com/georgysavva/dbscan/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

func ExampleQueryAll() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var users []*User
	if err := pgxscan.QueryAll(
		ctx, &users, db, `SELECT id, name, email, age from users`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// users variable now contains data from all rows.
}

func ExampleQueryOne() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var user User
	if err := pgxscan.QueryOne(
		ctx, &user, db, `SELECT id, name, email, age from users where id='bob'`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// users variable now contains data from all rows.
}

func ExampleScanAll() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	// Query pgx.Rows from the database.
	db, _ := pgxpool.Connect(ctx, "example-connection-url")
	rows, _ := db.Query(ctx, `SELECT id, name, email, age from users`)

	var users []*User
	if err := pgxscan.ScanAll(&users, rows); err != nil {
		// Handle rows processing error.
	}
	// users variable now contains data from all rows.
}

func ExampleScanOne() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	// Query pgx.Rows from the database.
	db, _ := pgxpool.Connect(ctx, "example-connection-url")
	rows, _ := db.Query(ctx, `SELECT id, name, email, age from users where id='bob'`)

	var user User
	if err := pgxscan.ScanOne(&user, rows); err != nil {
		// Handle rows processing error.
	}
	// user variable now contains data from the single row.
}

func ExampleRowScanner() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	// Query pgx.Rows from the database.
	db, _ := pgxpool.Connect(ctx, "example-connection-url")
	rows, _ := db.Query(ctx, `SELECT id, name, email, age from users`)
	// Make sure rows are closed.
	defer rows.Close()

	rs := pgxscan.NewRowScanner(rows)
	for rows.Next() {
		var user User
		if err := rs.Scan(&user); err != nil {
			// Handle row scanning error.
		}
		// user variable now contains data from the current row.
	}
	if err := rows.Err(); err != nil {
		// Handle rows final error.
	}
}

func ExampleRowScan() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	// Query pgx.Rows from the database.
	db, _ := pgxpool.Connect(ctx, "example-connection-url")
	rows, _ := db.Query(ctx, `SELECT id, name, email, age from users`)
	// Make sure rows are closed.
	defer rows.Close()

	for rows.Next() {
		var user User
		if err := pgxscan.ScanRow(&user, rows); err != nil {
			// Handle row scanning error.
		}
		// user variable now contains data from the current row.
	}
	if err := rows.Err(); err != nil {
		// Handle rows final error.
	}
}
