package dbscan_test

import (
	"database/sql"
	"github.com/georgysavva/dbscan"
)

type User3 struct {
	ID    string
	Name  string
	Email string
	Age   int
}

func ExampleScanAll() {
	// package

	// Query rows from the database that implement dbscan.Rows interface, e.g. *sql.Rows:
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT id, name, email, age from users`)

	var users []*User3
	if err := dbscan.ScanAll(&users, rows); err != nil {
		// Handle rows processing error
	}
	// users variable now contains data from all rows.
}
