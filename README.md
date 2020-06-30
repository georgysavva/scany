# dbscan

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/georgysavva/dbscan)
[![Build Status](https://travis-ci.com/georgysavva/dbscan.svg?branch=master)](https://travis-ci.com/georgysavva/dbscan) 
[![codecov](https://codecov.io/gh/georgysavva/dbscan/branch/master/graph/badge.svg)](https://codecov.io/gh/georgysavva/dbscan)
[![Go Report Card](https://goreportcard.com/badge/github.com/georgysavva/dbscan)](https://goreportcard.com/report/github.com/georgysavva/dbscan)

Library for scanning data from database into Go structs and more.

## Overview

Go favors simplicity and it's pretty common to work with database via driver directly without any ORM.
It provides great control and efficiency in your queries, but here is a problem: 
you need to manually iterate over database rows and scan data from all columns into a corresponding destination.
It can be error prone, verbose and just tedious. 

`dbscan` library aims to solve this problem, 
it allows developers to scan complex data from database into Go structs and other composite types 
with just one function call and don't bother with rows iteration.
It's not limited to any specific database, it works with standard `database/sql` library, 
so any database with `database/sql` driver is supported. 
It also supports pgx library specific for PostgreSQL. 

This library consists of the following packages: sqlscan, pgxscan and dbscan. 


## How to use with database/sql

```
type User struct {
    ID    string `db:"user_id"`
    Name  string
    Email string
    Age   int
}

// Query rows from the database that implement dbscan.Rows interface, e.g. *sql.Rows:
db, _ := sql.Open("pgx", "example-connection-url")
rows, _ := db.Query(`SELECT user_id, name, email, age from users`)

var users []*User
if err := dbscan.ScanAll(&users, rows); err != nil {
    // Handle rows processing error
}
// users variable now contains data from all rows.
```

## How to use with pgx

```
type User struct {
    ID    string `db:"user_id"`
    Name  string
    Email string
    Age   int
}

// Query rows from the database that implement dbscan.Rows interface, e.g. *sql.Rows:
db, _ := sql.Open("pgx", "example-connection-url")
rows, _ := db.Query(`SELECT user_id, name, email, age from users`)

var users []*User
if err := dbscan.ScanAll(&users, rows); err != nil {
    // Handle rows processing error
}
// users variable now contains data from all rows.
```

## Install

```
go get github.com/georgysavva/dbscan
```

## Tests

The only thing you need to run tests locally is an internet connection, 
it's required to download and cache the database binary.
Just type `go test ./...` inside dbscan root directory and let the code to the rest. 

## what it is not 

## Contributing 

Every feature request or question is really appreciated. Don't hesitate, just post an issue or PR.

## Roadmap   

Customize

## Supported Go versions 

dbscan supports Go 1.13 and higher.


## Versions policy

todo

## License

This project is licensed under the terms of the MIT license.
