package dbscan_test

import (
	dbscan2 "github.com/georgysavva/dbscan/dbscan"
)

func ExampleScanAll() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan2.Rows

	var users []*User
	if err := dbscan2.ScanAll(&users, rows); err != nil {
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

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan2.Rows

	var user User
	if err := dbscan2.ScanOne(&user, rows); err != nil {
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

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan2.Rows

	// Make sure rows are always closed.
	defer rows.Close()
	rs := dbscan2.NewRowScanner(rows)
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

func ExampleScanRow() {
	type User struct {
		ID    string
		Name  string
		Email string
		Age   int
	}

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan2.Rows

	// Make sure rows are always closed.
	defer rows.Close()
	for rows.Next() {
		var user User
		if err := dbscan2.ScanRow(&user, rows); err != nil {
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
