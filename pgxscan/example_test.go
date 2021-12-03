package pgxscan_test

import (
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/georgysavva/scany/dbscan"

	"github.com/georgysavva/scany/pgxscan"
)

func ExampleSelect() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var users []*User
	if err := pgxscan.Select(
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

	api, _ := getAPI()

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var users []*User
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

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var user User
	if err := pgxscan.Get(
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

	api, _ := getAPI()

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var user User
	if err := api.GetNamed(
		ctx, db, &user, `SELECT full_name, email, age FROM users WHERE user_id=:user_id`, &User{ID: "bob"},
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

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	user := &User{
		ID:       "billy",
		FullName: "Billy Bob",
		Email:    "billy@example.com",
		Age:      50,
	}

	api, _ := getAPI()

	if _, err := api.ExecNamed(
		ctx, db, `INSERT INTO users (full_name, email, age, user_id) VALUES (:full_name, :email, :age, :user_id)`, user,
	); err != nil {
		// Handle exec processing error.
	}
	// user has now been inserted into the users table

	// let us now delete it
	if _, err := api.ExecNamed(
		ctx, db, `DELETE FROM users WHERE user_id = :ID`, user,
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

	// Query pgx.Rows from the database.
	db, _ := pgxpool.Connect(ctx, "example-connection-url")
	rows, _ := db.Query(ctx, `SELECT user_id, full_name, email, age FROM users`)

	var users []*User
	if err := pgxscan.ScanAll(&users, rows); err != nil {
		// Handle rows processing error.
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

	// Query pgx.Rows from the database.
	db, _ := pgxpool.Connect(ctx, "example-connection-url")
	rows, _ := db.Query(ctx, `SELECT user_id, full_name, email, age FROM users WHERE user_id='bob'`)

	var user User
	if err := pgxscan.ScanOne(&user, rows); err != nil {
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

	// Query pgx.Rows from the database.
	db, _ := pgxpool.Connect(ctx, "example-connection-url")
	rows, _ := db.Query(ctx, `SELECT user_id, full_name, email, age FROM users`)
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

func ExampleScanRow() {
	type User struct {
		ID       string `db:"user_id"`
		FullName string
		Email    string
		Age      int
	}

	// Query pgx.Rows from the database.
	db, _ := pgxpool.Connect(ctx, "example-connection-url")
	rows, _ := db.Query(ctx, `SELECT user_id, full_name, email, age FROM users`)
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

// This example shows how to create and use a custom API instance to override default settings.
func ExampleAPI() {
	type User struct {
		ID       string `database:"userid"`
		FullName string
		Email    string
		Age      int
	}

	// Instantiate a custom API with overridden settings.
	dbscanAPI, err := pgxscan.NewDBScanAPI(
		dbscan.WithFieldNameMapper(strings.ToLower),
		dbscan.WithStructTagKey("database"),
	)
	if err != nil {
		// Handle dbscan API initialization error.
	}
	api, err := pgxscan.NewAPI(dbscanAPI)
	if err != nil {
		// Handle pgxscan API initialization error.
	}

	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var users []*User
	// Use the custom API instance to access pgxscan functionality.
	if err := api.Select(
		ctx, db, &users, `SELECT userid, fullname, email, age FROM users`,
	); err != nil {
		// Handle query or rows processing error.
	}
	// users variable now contains data from all rows.
}
