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

func ExampleSelectNamed() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	type Table struct {
		Name string
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	var users []*User

	api, err := getAPI()
	if err != nil {
		return
	}

	if err := api.SelectNamed(
		ctx, db, &users, `SELECT user_id, full_name, email, age FROM :name`, &Table{Name: "users"},
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

	api, err := getAPI()
	if err != nil {
		return
	}

	var user User
	if err := api.Get(
		ctx, db, &user, `SELECT user_id, full_name, email, age FROM users WHERE user_id='bob'`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// user variable now contains data from all rows.
}

func ExampleGetNamed() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	api, err := getAPI()
	if err != nil {
		return
	}

	var user User
	if err := api.GetNamed(
		ctx, db, &user, `SELECT full_name, email, age FROM users WHERE user_id = :user_id`, &User{ID: "bob"},
	); err != nil {
		// Handle query or rows processing error.
	}
	// user variable now contains data from all rows.
}

func ExampleExecNamed() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	db, _ := sql.Open("postgres", "example-connection-url")

	user := &User{
		ID:       "billy",
		FullName: "Billy Bob",
		Email:    "billy@example.com",
		Age:      50,
	}

	api, err := getAPI()
	if err != nil {
		return
	}

	if _, err := api.ExecNamed(
		ctx, db, `INSERT INTO users (full_name, email, age, user_id) VALUES (:full_name, :email, :age, :user_id)`, user,
	); err != nil {
		// Handle exec processing error.
	}
	// user has now been inserted into the users table

	// let us now delete it
	if _, err := api.ExecNamed(
		ctx, db, `DELETE FROM users WHERE user_id = :user_id`, user,
	); err != nil {
		// Handle exec processing error.
	}
	// user has now been deleted from the database
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
