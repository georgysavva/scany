---
name: New Issue template
about: This template helps new contributors to create more effective issues.
title: ''
labels: ''
assignees: ''

---

Thank you for your interest in the project!

Before opening a new issue, please make sure that you have completed all of the following steps: 
- [ ] I read the documentation to the `dbscan` package: https://pkg.go.dev/github.com/georgysavva/scany/dbscan
- [ ] I read the documentation to `pgxscan` package (if you work with `pgx` library): https://pkg.go.dev/github.com/georgysavva/scany/pgxscan
- [ ] I read the documentation to the `sqlscan` package (if you work with `database/sql` library): https://pkg.go.dev/github.com/georgysavva/scany/sqlscan
- [ ] I searched for already existing similar issues on Github: https://github.com/georgysavva/scany/issues

To help with debugging please provide your code with all relevant information to scany (if applicable). 
The code should contain:
- The definition of the types that you pass to scany library.
- Your SQL query.
- How you interact with scany library.

Example of the code you could provide to aid debugging:
```go
type User struct {
    Name string
}

sqlQuery := "SELECT name from users"

var users []*User
err := pgxscan.Select(ctx, db, &users, sqlQuery) 
```
