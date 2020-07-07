// Package scany is a set of packages for scanning data from a database into Go structs and more.
/*
scany isn't limited to any specific database. It integrates with standard database/sql library,
so any database with database/sql driver is supported.
It also works with https://github.com/jackc/pgx - specific library for PostgreSQL.
Apart from the out of the box support, scany can be easily extended to work with any database library.

scany contains the following packages:

sqlscan package works with database/sql standard library.

pgxscan package works with github.com/jackc/pgx/v4 library.

dbscan package works with an abstract database and can be integrated with any library.
This particular package implements core scany features and contains all the logic.
Both sqlscan and pgxscan use dbscan internally.
*/
package scany
