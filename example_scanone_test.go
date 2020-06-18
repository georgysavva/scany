package dbscan_test

import (
	"database/sql"

	"github.com/georgysavva/dbscan"
)

func Example_scanOne() {
	// package
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
