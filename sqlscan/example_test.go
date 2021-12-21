package sqlscan_test

import (
	"database/sql"
	"strings"

	"github.com/georgysavva/scany/dbscan"
	"github.com/georgysavva/scany/sqlscan"
)

func ExampleSelect() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	var users []*User
	if err := sqlscan.Select(
		ctx, db, &users, `SELECT user_id, full_name, email, age FROM users`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// users variable now contains data from all rows.
}

func ExampleGet() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	var user User
	if err := sqlscan.Get(
		ctx, db, &user, `SELECT user_id, full_name, email, age FROM users WHERE user_id='bob'`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// user variable now contains data from all rows.
}

func ExampleScanAll() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	// Query *sql.Rows from the database.
	db, _ := sql.Open("postgres", "example-connection-url")
	rows, _ := db.Query(`SELECT user_id, full_name, email, age FROM users`)

	var users []*User
	if err := sqlscan.ScanAll(&users, rows); err != nil {
		// Handle rows processing error
	}
	// users variable now contains data from all rows.
}

func ExampleScanOne() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	// Query *sql.Rows from the database.
	db, _ := sql.Open("postgres", "example-connection-url")
	rows, _ := db.Query(`SELECT user_id, full_name, email, age FROM users WHERE user_id='bob'`)

	var user User
	if err := sqlscan.ScanOne(&user, rows); err != nil {
		// Handle rows processing error.
	}
	// user variable now contains data from the single row.
}

func ExampleRowScanner() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	// Query *sql.Rows from the database.
	db, _ := sql.Open("postgres", "example-connection-url")
	rows, _ := db.Query(`SELECT user_id, full_name, email, age FROM users`)
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
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	// Query *sql.Rows from the database.
	db, _ := sql.Open("postgres", "example-connection-url")
	rows, _ := db.Query(`SELECT user_id, full_name, email, age FROM users`)
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

// This example shows how to create and use a custom API instance to override default settings.
func ExampleAPI() {
	type User struct {
		ID       string `database:"userid"`
		FullName string
		Email    string
		Age      int
	}

	// Instantiate a custom API with overridden settings.
	dbscanAPI, err := sqlscan.NewDBScanAPI(
		dbscan.WithFieldNameMapper(strings.ToLower),
		dbscan.WithStructTagKey("database"),
	)
	if err != nil {
		// Handle dbscan API initialization error.
	}
	api, err := sqlscan.NewAPI(dbscanAPI)
	if err != nil {
		// Handle sqlscan API initialization error.
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	var users []*User
	// Use the custom API instance to access sqlscan functionality.
	if err := api.Select(
		ctx, db, &users, `SELECT userid, fullname, email, age FROM users`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// users variable now contains data from all rows.
}
