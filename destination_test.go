package dbscan_test

import (
	"reflect"
	"testing"

	"github.com/georgysavva/dbscan"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDestination_ValidDst_ReturnsElemReflectValue(t *testing.T) {
	t.Parallel()
	var dst struct{ Foo string }
	expected := reflect.ValueOf(&dst).Elem()

	got, err := dbscan.ParseDestination(&dst)
	require.NoError(t, err)

	assert.Equal(t, expected, got)
}

func TestParseDestination_InvalidDst_ReturnsErr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		dst         interface{}
		expectedErr string
	}{
		{
			name: "non pointer",
			dst: struct {
				Foo string
			}{},
			expectedErr: "dbscan: destination must be a pointer, got: struct { Foo string }",
		},
		{
			name:        "map",
			dst:         map[string]interface{}{},
			expectedErr: "dbscan: destination must be a pointer, got: map[string]interface {}",
		},
		{
			name:        "slice",
			dst:         []struct{ Foo string }{},
			expectedErr: "dbscan: destination must be a pointer, got: []struct { Foo string }",
		},
		{
			name:        "nil",
			dst:         nil,
			expectedErr: "dbscan: destination must be a non nil pointer",
		},
		{
			name:        "(*int)(nil)",
			dst:         (*int)(nil),
			expectedErr: "dbscan: destination must be a non nil pointer",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := dbscan.ParseDestination(tc.dst)
			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}
