// Package dbscan allows scanning data from abstract database rows into Go structs and more.
/*
dbscan works with abstract Rows and doesn't depend on any specific database or a library.
If a type implements Rows interface it can leverage full functional of this package.

Scanning into struct

The main feature of dbscan is ability to scan row data into struct.

	type User struct {
		ID        string `db:"user_id"`
		FirstName string
		Email     string
	}

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan.Rows

	var users []*User
	dbscan.ScanAll(&users, rows)
	// users variable now contains data from all rows.

By default, to get the corresponding column dbscan translates field name to snake case.
In order to override this behaviour, specify column name in the `db` field tag.
In the example above User struct is mapped to the following columns: "user_id", "first_name", "email".

dbscan works recursively, struct can contain embedded structs as well.
It allows to reuse models in different queries. Structs can be embedded both by value and by a pointer.
Note that, nested non-embedded structs aren't allowed, this decision was made due to simplicity.
By default, dbscan maps fields from embedded structs to columns as is and doesn't add any prefix,
this simulates behaviour of major SQL databases in case of a JOIN.
In order to add a prefix to all fields of the embedded struct specify it in the `db` field tag,
dbscan uses "." as a separator, for example:

	type User struct {
		UserID string
		Email  string
	}

	type Post struct {
		ID   string
		Text string
	}

	type Row struct {
		*User
		Post `db:"post"`
	}

Row struct is mapped to the following columns: "user_id", "email", "post.id", "post.text".

In order for dbscan to work with a field it must be exported, unexported fields will be ignored.
This applied to embedded structs too, the type that is embedded must be exported.

It's possible to explicitly mark a field as ignored for dbscan. To do this set `db:"-"` struct tag.
By the way, it also works for embedded structs as well, for example:

	type Post struct {
		ID   string
		Text string
	}

	type Comment struct {
		Post  `db:"-"`
		ID    string
		Body  string
		Likes int `db:"-"`
	}

Comment struct is mapped to the following columns: "id", "body".

In case there is no corresponding field for a column dbscan returns an error,
this forces to only select data from the database that application needs. And another way around,
if a struct contains multiple fields that are mapped to the same column,
dbscan won't be able to make the chose to which field to assign and will return an error, for example:

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

Row struct is invalid since both Row.User.ID and Row.Post.ID are mapped to the "id" column.

Scanning into map

Apart from scanning into structs, dbscan can handle maps,
in that case it uses column name as the map key and column data as the map value, for example:

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan.Rows

	var results []map[string]interface{}
	dbscan.ScanAll(&results, rows)
	// results variable now contains data from all rows.

Map type isn't limited to map[string]interface{},
it can be any map with string key, e.g. map[string]string or map[string]int,
if all column values have the same specific type.

Scanning into other types

If the destination isn't a struct nor a map, dbscan handles it as a single column scan,
dbscan ensures that rows contain exactly one column and scans destination from that column, for example:

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan.Rows

	var results []string
	dbscan.ScanAll(&results, rows)
	// results variable not contains data from all single columns rows.

Duplicate columns

Rows must not contain duplicate columns, otherwise dbscan won't be able to decide
from which column to select and will return an error.

Rows processing

ScanAll and ScanOne functions take care of rows processing,
they iterate rows to the end and close them after that.
Client code doesn't need to bother with that, it just passes rows to dbscan.

Manual rows iteration

It's possible to manually control rows iteration, but still use all scanning features of dbscan,
see RowScanner for details.
*/
package dbscan
