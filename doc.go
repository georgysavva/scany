// Package scany is a set of packages for scanning data from database into Go structs and more.
/*
Go favors simplicity and it's pretty common to work with database via driver directly without any ORM.
It provides great control and efficiency in your queries, but here is a problem:
you need to manually iterate over database rows and scan data from all columns into a corresponding destination.
It can be error prone, verbose and just tedious.

scany library aims to solve this problem,
it allows developers to scan complex data from database into Go structs and other composite types
with just one function call and don't bother with rows iteration.
It's not limited to any specific database, it works with standard database/sql library,
so any database with database/sql driver is supported.
It also supports pgx library specific for PostgreSQL.

This library consists of the following packages: sqlscan, pgxscan and dbscan.
*/
package scany
