# scany

[![Tests Status](https://github.com/georgysavva/scany/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/georgysavva/scany/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/georgysavva/scany)](https://goreportcard.com/report/github.com/georgysavva/scany)
[![codecov](https://codecov.io/gh/georgysavva/scany/branch/master/graph/badge.svg)](https://codecov.io/gh/georgysavva/scany)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/georgysavva/scany/v2)](https://pkg.go.dev/github.com/georgysavva/scany/v2)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

## Overview

Go favors simplicity, and it's pretty common to work with a database via driver directly without any ORM. It provides
great control and efficiency in your queries, but here is a problem:
you need to manually iterate over database rows and scan data from all columns into a corresponding destination. It can
be error-prone verbose and just tedious. scany aims to solve this problem. It allows developers to scan complex data
from a database into Go structs and other composite types with just one function call and don't bother with rows
iteration.

scany isn't limited to any specific database. It integrates with `database/sql`, so any database with `database/sql`
driver is supported. It also works with [`pgx`](https://github.com/jackc/pgx) library native interface. Apart from the
out-of-the-box support, scany can be easily extended to work with almost any database library.

Note that scany isn't an ORM. First of all, it works only in one direction:
it scans data into Go objects from the database, but it can't build database queries based on those objects. Secondly,
it doesn't know anything about relations between objects e.g: one to many, many to many.

## Features

- Custom database column name via struct tag
- Reusing structs via nesting or embedding
- NULLs and custom types support
- Omitted struct fields
- Apart from structs, support for maps and Go primitive types as the destination
- Override default settings

## Install

```
go get github.com/georgysavva/scany/v2
```

## How to use with `database/sql`

```go
package main

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/v2/sqlscan"
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

Use [`sqlscan`](https://pkg.go.dev/github.com/georgysavva/scany/v2/sqlscan)
package to work with `database/sql` standard library.

## How to use with `pgx` native interface

```go
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/georgysavva/scany/v2/pgxscan"
)

type User struct {
	ID    string
	Name  string
	Email string
	Age   int
}

func main() {
	ctx := context.Background()
	db, _ := pgxpool.New(ctx, "example-connection-url")

	var users []*User
	pgxscan.Select(ctx, db, &users, `SELECT id, name, email, age FROM users`)
	// users variable now contains data from all rows.
}
```

Use [`pgxscan`](https://pkg.go.dev/github.com/georgysavva/scany/v2/pgxscan)
package to work with `pgx` library native interface.

## How to use with other database libraries

Use [`dbscan`](https://pkg.go.dev/github.com/georgysavva/scany/v2/dbscan) package that works with an abstract database,
and can be integrated with any library that has a concept of rows. This particular package implements core scany
features and contains all the logic. Both `sqlscan` and `pgxscan` use `dbscan` internally.

## Comparison with [`sqlx`](https://github.com/jmoiron/sqlx)

- sqlx only works with `database/sql` standard library. scany isn't limited to `database/sql`. It also
  supports [`pgx`](https://github.com/jackc/pgx) native interface and can be extended to work with any database library
  independent of `database/sql`
- In terms of scanning and mapping abilities, scany provides
  all [features](https://github.com/georgysavva/scany#features) of sqlx
- scany has a simpler API and much fewer concepts, so it's easier to start working with

## Project documentation

For detailed project documentation see GitHub [Wiki](https://github.com/georgysavva/scany/wiki).

## How to contribute

- If you have an idea or a question, just post a pull request or an issue. Every feedback is appreciated.
- If you want to help but don't know-how. All issues that you can work on are marked as `"help wanted"`. Discover all `"help wanted"` issues [here](https://github.com/georgysavva/scany/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22).

## License

This project is licensed under the terms of the MIT license.
