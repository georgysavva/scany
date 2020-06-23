// Package dbscan allows scanning data from database rows into complex Go types.
/*
dbscan works with abstract Rows and doesn't depend on any specific database or library.
If a type implements Rows interface it can leverage full functional of this package.
Subpackages sqlscan and pgxscan are wrappers around this package
they contain functions and adapters tailored to database/sql and
github.com/jackc/pgx/v4 libraries correspondingly. sqlscan and pgxscan proxy all calls to dbscan internally.
dbscan does all the logic, but generally, it shouldn't be imported by the application code directly.

If you are working with database/sql - use sqlscan subpackage.

If you are working with pgx - use pgxscan subpackage.

Scanning into struct

The main feature of dbscan is ability to scan row data into struct.

	type User struct {
		ID        string `db:"user_id"`
		FirstName string
		Email     string
	}

	var users []*User
	if err := dbscan.ScanAll(&users, rows); err != nil {
		// Handle rows processing error
	}
	// users variable now contains data from all rows.

By default, to get the corresponding column dbscan translates field name to snake case.
In order to override this behaviour, specify column name in the `db` field tag.
In the example above User struct is mapped to the following columns: "user_id", "first_name", "email".

Struct can contain embedded structs as well. It allows to reuse models in different queries.
Note that non-embedded structs aren't allowed, this decision was made due to simplicity.
By default, dbscan maps fields from embedded structs to columns as is and doesn't add prefix,
this simulates behaviour of major SQL databases in case of a JOIN.
In order to add a prefix to all fields of the embedded struct specify it in the `db` field tag,
"." used as the separator for example:

	type User struct {
		UserID    string
		Email     string
	}

	type Post struct {
		ID   string
		Text string
	}

	type Row struct {
		User
		Post `db:post`
	}

will get mapped to the following columns: "user_id", "email", "post.id", "post.text".

If dbscan can't find corresponding field for a column it returns an error,
this forces to only select data from the database that application needs.
Also if struct contains multiple fields that are mapped to the same column,
dbscan won't be able to make the chose to which field to assign and return an error, for example:

	type User struct {
		ID    string
		Email string
	}

	type Post struct {
		ID   string
		Text string
	}

	type Row struct {
		User
		Post
	}

Row struct is invalid since both User.ID and Post.ID are mapped to the "id" column.

Scanning into map

Apart from scanning into structs, dbscan can handle maps,
in that case it uses column name as the map key and column data as the map value. For example:

	var results []map[string]interface{}
	if err := dbscan.ScanAll(&result, rows); err != nil {
		// Handle rows processing error
	}
	// results variable now contains data from the row.

Note that map type isn't limited to map[string]interface{},
it can be any map with string key, e.g. map[string]string or map[string]int,
if all column values have the same specific type.

Scanning into other types

If the destination isn't a struct nor a map, dbscan handles it as single column scan,
it ensures that rows contain exactly one column and scans destination from the column, for example:

	var result []string
	if err := dbscan.ScanAll(&result, rows); err != nil {
		// Handle rows processing error
	}
	// result variable not contains data from the row single column.

Rows processing

ScanAll and ScanOne functions take care of rows processing,
they iterate rows to the end and close them after that.
Client code doesn't need bother with all of that, it just needs to pass rows to dbscan.
*/
package dbscan
