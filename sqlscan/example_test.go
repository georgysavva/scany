package sqlscan_test

import (
	"database/sql"

	"github.com/georgysavva/scany/sqlscan"
)

func ExampleQueryAll() {
	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	db, _ := sql.Open("pgx", "example-connection-url")

	var users []*User
	if err := sqlscan.QueryAll(
		ctx, &users, db, `SELECT user_id, name, email, age FROM users`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// users variable now contains data from all rows.
}

func ExampleQueryOne() {
	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	db, _ := sql.Open("pgx", "example-connection-url")

	var user User
	if err := sqlscan.QueryOne(
		ctx, &user, db, `SELECT user_id, name, email, age FROM users WHERE id='bob'`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// user variable now contains data from all rows.
}

func ExampleScanAll() {
	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	// Query *sql.Rows from the database.
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT user_id, name, email, age FROM users`)

	var users []*User
	if err := sqlscan.ScanAll(&users, rows); err != nil {
		// Handle rows processing error
	}
	// users variable now contains data from all rows.
}

func ExampleScanOne() {
	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	// Query *sql.Rows from the database.
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT user_id, name, email, age FROM users WHERE id='bob'`)

	var user User
	if err := sqlscan.ScanOne(&user, rows); err != nil {
		// Handle rows processing error.
	}
	// user variable now contains data from the single row.
}

func ExampleRowScanner() {
	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	// Query *sql.Rows from the database.
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT user_id, name, email, age FROM users`)

	// Make sure rows are always closed.
	defer rows.Close()
	rs := sqlscan.NewRowScanner(rows)
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

func ExampleScanRow() {
	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	// Query *sql.Rows from the database.
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT user_id, name, email, age FROM users`)

	// Make sure rows are always closed.
	defer rows.Close()
	for rows.Next() {
		var user User
		if err := sqlscan.ScanRow(&user, rows); err != nil {
			// Handle row scanning error.
		}
		// user variable now contains data from the current row.
	}
	if err := rows.Err(); err != nil {
		// Handle rows final error.
	}
}
