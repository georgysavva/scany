package dbscan_test

import (
	"database/sql"
	"github.com/georgysavva/dbscan"
)

type User2 struct {
	ID    string
	Name  string
	Email string
	Age   int
}

func ExampleRowScanner() {
	// Query rows from the database that implement dbscan.Rows interface, e.g. *sql.Rows:
	db, _ := sql.Open("pgx", "example-connection-url")
	rows, _ := db.Query(`SELECT id, name, email, age from users`)

	// Make sure rows are closed.
	defer rows.Close()

	rs := dbscan.NewRowScanner(rows)
	for rows.Next() {
		var user User2
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
