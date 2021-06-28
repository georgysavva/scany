---
name: New Issue template
about: This template helps new contributors to create more effective issues.
title: ''
labels: ''
assignees: ''

---

To help with debugging please provide your code with all relevant information to scany (if applicable). 
The code should contain:
- The definition of the types that you pass to scany library.
- Your SQL query.
- How you call scany API.

Example of the code you could provide to aid debugging:
```go
type User struct {
    Name string
}

sqlQuery := "SELECT name from users"

var users []*User
err := pgxscan.Select(ctx, db, &users, sqlQuery) 
```
