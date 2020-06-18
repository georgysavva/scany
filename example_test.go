package dbscan_test

import (
	"database/sql"
	"github.com/georgysavva/dbscan"
)

func ExampleScanAll() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	// Query rows from the database that implement dbscan.Rows interface, e.g. *sql.Rows:
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT id, name, email, age from users`)

	var users []*User
	if err := dbscan.ScanAll(&users, rows); err != nil {
		// Handle rows processing error
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

	// Query rows from the database that implement dbscan.Rows interface, e.g. *sql.Rows:
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT id, name, email, age from users where id='bob'`)

	var user User
	if err := dbscan.ScanOne(&user, rows); err != nil {
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

	// Query rows from the database that implement dbscan.Rows interface, e.g. *sql.Rows:
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT id, name, email, age from users`)

	// Make sure rows are closed.
	defer rows.Close()

	rs := dbscan.NewRowScanner(rows)
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
	if err := rows.Close(); err != nil {
		// Handle rows closing error.
	}
}

func ExampleRowScan() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	// Query rows from the database that implement dbscan.Rows interface, e.g. *sql.Rows:
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT id, name, email, age from users`)

	// Make sure rows are closed.
	defer rows.Close()

	for rows.Next() {
		var user User
		if err := dbscan.ScanRow(&user, rows); err != nil {
			// Handle row scanning error.
		}
		// user variable now contains data from the current row.
	}
	if err := rows.Err(); err != nil {
		// Handle rows final error.
	}
	if err := rows.Close(); err != nil {
		// Handle rows closing error.
	}
}
