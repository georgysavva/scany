package dbscan_test

import (
	"strings"

	"github.com/georgysavva/scany/dbscan"
)

func ExampleScanAll() {
	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	// Query rows from the database that implement Rows interface.
	var rows dbscan.Rows

	var users []*User
	if err := dbscan.ScanAll(&users, rows); err != nil {
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

	// Query rows from the database that implement Rows interface.
	var rows dbscan.Rows

	var user User
	if err := dbscan.ScanOne(&user, rows); err != nil {
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

	// Query rows from the database that implement Rows interface.
	// You should also take care of handling rows error after iteration and closing them.
	var rows dbscan.Rows

	rs := dbscan.NewRowScanner(rows)

	for rows.Next() {
		var user User
		if err := rs.Scan(&user); err != nil {
			// Handle row scanning error.
		}
		// user variable now contains data from the current row.
	}
}

func ExampleScanRow() {
	type User struct {
		ID    string `db:"user_id"`
		Name  string
		Email string
		Age   int
	}

	// Query rows from the database that implement Rows interface.
	// You should also take care of handling rows error after iteration and closing them.
	var rows dbscan.Rows

	for rows.Next() {
		var user User
		if err := dbscan.ScanRow(&user, rows); err != nil {
			// Handle row scanning error.
		}
		// user variable now contains data from the current row.
	}
}

// ExampleAPI shows how to create and use a custom API instance with overridden settings.
func ExampleAPI() {
	type User struct {
		ID    string `database:"user_id"`
		Name  string
		Email string
		Age   int
	}

	// Instantiate a custom API with overridden settings.
	api := dbscan.NewAPI(
		dbscan.WithFieldNameMapper(strings.ToLower),
		dbscan.WithStructTagKey("database"),
	)

	// Query rows from the database that implement Rows interface.
	// You should also take care of handling rows error after iteration and closing them.
	var rows dbscan.Rows

	var users []*User
	// Use the custom API instance to access dbscan functionality.
	if err := api.ScanAll(&users, rows); err != nil {
		// Handle rows processing error
	}
	// users variable now contains data from all rows.
}
