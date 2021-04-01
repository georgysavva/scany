# scany

[![Build Status](https://travis-ci.com/georgysavva/scany.svg?branch=master)](https://travis-ci.com/georgysavva/scany) 
[![Go Report Card](https://goreportcard.com/badge/github.com/georgysavva/scany)](https://goreportcard.com/report/github.com/georgysavva/scany)
[![codecov](https://codecov.io/gh/georgysavva/scany/branch/master/graph/badge.svg)](https://codecov.io/gh/georgysavva/scany)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/georgysavva/scany)](https://pkg.go.dev/github.com/georgysavva/scany)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)  

## Overview

Go favors simplicity, and it's pretty common to work with a database via driver directly without any ORM.
It provides great control and efficiency in your queries, but here is a problem: 
you need to manually iterate over database rows and scan data from all columns into a corresponding destination.
It can be error-prone verbose and just tedious. 
scany aims to solve this problem. 
It allows developers to scan complex data from a database into Go structs and other composite types 
with just one function call and don't bother with rows iteration.

scany isn't limited to any specific database. It integrates with `database/sql`, 
so any database with `database/sql` driver is supported. 
It also works with [pgx](https://github.com/jackc/pgx) library native interface. 
Apart from the out-of-the-box support, scany can be easily extended to work with almost any database library.

Note that scany isn't an ORM. First of all, it works only in one direction: 
it scans data into Go objects from the database, but it can't build database queries based on those objects.
Secondly, it doesn't know anything about relations between objects e.g: one to many, many to many.

## Features

* Custom database column name via struct tag
* Reusing structs via nesting or embedding 
* NULLs and custom types support
* Omitted struct fields
* Apart from structs, support for other destination types: maps, slices, etc.

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
	ID    string
	Name  string
	Email string
	Age   int
}

func main() {
	ctx := context.Background()
	db, _ := sql.Open("postgres", "example-connection-url")

	var users []*User
	sqlscan.Select(ctx, db, &users, `SELECT id, name, email, age FROM users`)
	// users variable now contains data from all rows.
}
```

Use [`sqlscan`](https://pkg.go.dev/github.com/georgysavva/scany/sqlscan) 
package to work with `database/sql` standard library. 


## How to use with `pgx` native interface

```go
package main

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/georgysavva/scany/pgxscan"
)

type User struct {
	ID    string
	Name  string
	Email string
	Age   int
}

func main() {
	ctx := context.Background()
	db, _ := pgxpool.Connect(ctx, "example-connection-url")

	var users []*User
	pgxscan.Select(ctx, db, &users, `SELECT id, name, email, age FROM users`)
	// users variable now contains data from all rows.
}
```

Use [`pgxscan`](https://pkg.go.dev/github.com/georgysavva/scany/pgxscan) 
package to work with `pgx` library native interface. 

## How to use with other database libraries

Use [`dbscan`](https://pkg.go.dev/github.com/georgysavva/scany/dbscan) package that works with an abstract database, 
and can be integrated with any library that has a concept of rows. 
This particular package implements core scany features and contains all the logic.
Both `sqlscan` and `pgxscan` use `dbscan` internally.

## Comparison with [sqlx](https://github.com/jmoiron/sqlx)

* sqlx only works with `database/sql` standard library. scany isn't limited only to `database/sql`. 
  It also supports [pgx](https://github.com/jackc/pgx) native interface and can be extended to work with any database library independent of `database/sql`
* In terms of scanning and mapping abilities, scany provides all [features](https://github.com/georgysavva/scany#features) of sqlx
* scany has a simpler API and much fewer concepts, so it's easier to start working with

## Supported Go versions 

scany supports Go 1.13 and higher.

## Roadmap   

* Add ability to set custom function to translate struct field to the column name, 
instead of the default to snake case function 
* Allow to use a custom separator for embedded structs prefix, instead of the default "."

## Tests

The easiest way to run the tests is:
```
go test ./...
``` 
scany runs a CockroachDB server to execute its tests.
It will download, cache and run the CockroachDB binary for you.
It's very convenient since the only requirement to run the tests is an internet connection. 
Alternatively, 
you can [download](https://www.cockroachlabs.com/docs/v20.2/install-cockroachdb-mac) the CockroachDB binary yourself 
and pass the path to the binary into tests: 
```
go test ./... -cockroach-binary cockroach
```

## golangci-lint

This project uses `golangci-lint` v1.38.0.

To run the linter locally do the following:
1. [Install](https://golangci-lint.run/usage/install/) `golangci-lint` program
2. In the project root type: `golangci-lint run`

## Contributing 

Every feature request or question is appreciated. Don't hesitate. Just post an issue or PR.

## License

This project is licensed under the terms of the MIT license.
