# Scany

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/georgysavva/scany)
[![Build Status](https://travis-ci.com/georgysavva/scany.svg?branch=master)](https://travis-ci.com/georgysavva/scany) 
[![codecov](https://codecov.io/gh/georgysavva/scany/branch/master/graph/badge.svg)](https://codecov.io/gh/georgysavva/scany)
[![Go Report Card](https://goreportcard.com/badge/github.com/georgysavva/scany)](https://goreportcard.com/report/github.com/georgysavva/scany)

Library for scanning data from a database into Go structs and more.

## Overview

Go favors simplicity, and it's pretty common to work with a database via driver directly without any ORM.
It provides great control and efficiency in your queries, but here is a problem: 
you need to manually iterate over database rows and scan data from all columns into a corresponding destination.
It can be error prone, verbose and just tedious. 
Scany aims to solve this problem, 
it allows developers to scan complex data from a database into Go structs and other composite types 
with just one function call and don't bother with rows iteration.

Scany isn't limited to any specific database. It integrates with standard `database/sql` library, 
so any database with `database/sql` driver is supported. 
It also works with [pgx](https://github.com/jackc/pgx) - specific library for PostgreSQL. 
Apart from supporting `database/sql` and `pgx` out of the box, 
Scany can be easily extended to work with any database library.

## Install

```
go get github.com/georgysavva/scany
```

## How to use with `database/sql`

```go
package main

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/sqlscan"
)

type User struct {
	UserID string
	Name   string
	Email  string
	Age    int
}

func main() {
	ctx := context.Background()
	db, _ := sql.Open("postgres", "example-connection-url")

	var users []*User
	sqlscan.Query(ctx, &users, db, `SELECT user_id, name, email, age FROM users`)
	// users variable now contains data from all rows.
}
```

Use [`sqlscan`](https://pkg.go.dev/github.com/georgysavva/scany/sqlscan) 
package to work with `database/sql` standard library. 


## How to use with `pgx`

```go
package main

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/georgysavva/scany/pgxscan"
)

type User struct {
	UserID string
	Name   string
	Email  string
	Age    int
}

func main() {
	ctx := context.Background()
	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var users []*User
	pgxscan.Query(ctx, &users, db, `SELECT user_id, name, email, age FROM users`)
	// users variable now contains data from all rows.
}
```

Use [`pgxscan`](https://pkg.go.dev/github.com/georgysavva/scany/pgxscan) 
package to work with `pgx` library. 

## How to use with other database libraries

Use [`dbscan`](https://pkg.go.dev/github.com/georgysavva/scany/dbscan) package that works with an abstract database, 
and can be integrated with any library. 
This particular package implements core scany features and contains all the logic.
Both `sqlscan` and `pgxscan` use `dbscan` internally.

## Supported Go versions 

Scany supports Go 1.13 and higher.

## Roadmap   

Customize

## Tests

The only thing you need to run tests locally is an internet connection, 
it's required to download and cache the database binary.
Just type `go test ./...` inside scany root directory and let the code do the rest. 

## Contributing 

Every feature request or question is really appreciated. Don't hesitate, just post an issue or PR.

## License

This project is licensed under the terms of the MIT license.
