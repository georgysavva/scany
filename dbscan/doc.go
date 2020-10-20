// Package dbscan allows scanning data from abstract database rows into Go structs and more.
/*
dbscan works with abstract Rows interface and doesn't depend on any specific database or a library.
If a type implements Rows it can leverage the full functionality of this package.

Mapping struct field to database column

The main feature of dbscan is the ability to scan rows data into structs.

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

By default, to get the corresponding database column dbscan translates struct field name to snake case.
To override this behavior, specify the column name in the `db` field tag.
In the example above User struct is mapped to the following columns: "user_id", "first_name", "email".

If selected rows contain a column that doesn't have a corresponding struct field dbscan returns an error,
this forces to only select data from the database that application needs.

Reusing structs

dbscan works recursively, a struct can contain embedded or nested structs as well.
It allows reusing models in different queries. Structs can be embedded or nested both by value and by a pointer.
If you don't specify the `db` tag, dbscan maps fields from nested structs to database columns
with the struct field name translated to snake case as the prefix,
on the opposite, fields from embedded structs are mapped to database columns without any prefix.
dbscan uses "." to separate the prefix. Here is an example:

	type UserPost struct {
		*User
		Post Post
	}

	type User struct {
		UserID string
		Email  string
	}

	type Post struct {
		ID   string
		Text string
	}

UserPost struct is mapped to the following columns: "user_id", "email", "post.id", "post.text".

To add a prefix to an embedded struct or change the prefix of a nested struct specify it in the `db` field tag.
You can also use the empty tag `db:""` to remove the prefix of a nested struct. Here is an example:

	type UserPostComment struct {
		*User   `db:"user"`
		Post    Post    `db:"p"`
		Comment Comment `db:""`
	}

	type User struct {
		UserID string
		Email  string
	}

	type Post struct {
		ID   string
		Text string
	}

	type Comment struct {
		CommentBody string
	}

UserPostComment struct is mapped to the following columns:
"user.user_id", "user.email", "p.id", "p.text", "comment_body".

NULLs and custom types

dbscan supports custom types and NULLs perfectly.
You can work with them the same way as if you would be using your database library directly.
Under the hood, dbscan passes all types that you provide to the underlying rows.Scan()
and if the database library supports a type, dbscan supports it automatically, for example:

	type User struct {
		OptionalBio  *string
		OptionalAge  CustomNullInt
		Data         CustomData
		OptionalData *CustomData
	}

	type CustomNullInt struct {
		// Any fields that this custom type needs
	}

	type CustomData struct {
		// Any fields that this custom type needs
	}

User struct is valid and every field will be scanned properly, the only condition for this
is that your database library can handle *string, CustomNullInt, CustomData and *CustomData types.

Ignored struct fields

In order for dbscan to work with a field it must be exported, unexported fields will be ignored.
The only exception is embedded structs, the type that is embedded might be unexported.

It's possible to explicitly mark a field as ignored for dbscan. To do this set `db:"-"` struct tag.
By the way, it works for nested and embedded structs as well, for example:

	type Comment struct {
		Post  `db:"-"`
		ID    string
		Body  string
		Likes int `db:"-"`
	}

	type Post struct {
		ID   string
		Text string
	}

Comment struct is mapped to the following columns: "id", "body".

Ambiguous struct fields

If a struct contains multiple fields that are mapped to the same database column,
dbscan will assign to the outermost and topmost field, for example:

	type UserPost struct {
		User
		Post
	}

	type Post struct {
		PostID string
		Text   string
		UserID string
	}

	type User struct {
		UserID string
		Email  string
	}

UserPost struct is mapped to the following columns: "user_id", "email", "post_id", "text".
But both UserPost.User.UserID and UserPost.Post.UserID are mapped to the "user_id" column,
since the User struct is embedded above the Post struct in the UserPost type,
UserPost.User.UserID will receive data from the "user_id" column and UserPost.Post.UserID will remain empty.
Note that you can't access it as UserPost.UserID though. it's an error for Go, and
you need to use the full version: UserPost.User.UserID

Scanning into map

Apart from scanning into structs, dbscan can handle maps,
in that case, it uses database column name as the map key and column data as the map value, for example:

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan.Rows

	var results []map[string]interface{}
	dbscan.ScanAll(&results, rows)
	// results variable now contains data from all rows.

Map type isn't limited to map[string]interface{},
it can be any map with a string key, e.g. map[string]string or map[string]int,
if all column values have the same specific type.

Scanning into other types

If the destination isn't a struct nor a map, dbscan handles it as a single column scan,
dbscan ensures that rows contain exactly one column and scans destination from that column, for example:

	// Query rows from the database that implement dbscan.Rows interface.
	var rows dbscan.Rows

	var results []string
	dbscan.ScanAll(&results, rows)
	// results variable not contains data from all rows single column.

Duplicate columns

Rows must not contain duplicate columns otherwise dbscan won't be able to decide
from which column to select and will return an error.

Rows processing

ScanAll and ScanOne functions take care of rows processing,
they iterate rows to the end and close them after that.
Client code doesn't need to bother with that, it just passes rows to dbscan.

Manual rows iteration

It's possible to manually control rows iteration but still use all scanning features of dbscan,
see RowScanner for details.

Implementing Rows interface

dbscan can be used with any database library that has a concept of rows and can implement dbscan Rows interface.
It's pretty likely that your rows type already implements Rows interface as-is,
for example this is true for the standard *sql.Rows type.
Or you just need a thin adapter how it was done for pgx.Rows in pgxscan, see pgxscan.RowsAdapter for details.
*/
package dbscan
