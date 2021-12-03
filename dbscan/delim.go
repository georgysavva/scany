package dbscan

import (
	"strings"
)

//DriverDelim is type of function accepted by the internal lexer for appending named arguments in the format of the native driver
type DriverDelim = func(builder *strings.Builder, index uint8) error

//SequentialDollarDelim delimeter format is used by most Postgres database client drivers natively
func SequentialDollarDelim(builder *strings.Builder, index uint8) error {
	_, err := builder.WriteRune('$')
	if err != nil {
		return err
	}

	return builder.WriteByte('0' + index)
}

// QuestionDelim delimeter format is used by most MySQL database client drivers natively
func QuestionDelim(builder *strings.Builder, index uint8) error {
	_, err := builder.WriteRune('?')
	return err
}
