package dbscan

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Level1 struct {
	Level2
	Foo string
	Sub Sub
}

type Sub struct {
	Foo string
}

type Level2 struct {
	Level3
}

type Level3 struct {
	Level4
}

type Level4 struct {
	Zero  string
	One   string
	Two   string `db:"-"`
	Three string `db:"not_three"`
}

func TestFieldIndex(t *testing.T) {
	s := Level1{}
	m := getColumnToFieldIndexMap(reflect.TypeOf(s))

	assert.Equal(t, []int{0, 0, 0, 0}, m["zero"])
	assert.Equal(t, []int{0, 0, 0, 1}, m["one"])
	assert.Equal(t, []int{0, 0, 0, 3}, m["not_three"])
	assert.Equal(t, []int{1}, m["foo"])
	_, found := m["two"]
	assert.Falsef(t, found, "column two should be absent")
	assert.Equal(t, []int{2, 0}, m["sub.foo"])
}
