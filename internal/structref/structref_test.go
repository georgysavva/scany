package structref

import (
	"reflect"
	"testing"
	"time"
)

func TestGetColumnToFieldIndexMap(t *testing.T) {
	type NestedTheme struct {
		ID   string
		Name string
	}
	type Embed struct {
		Play bool
	}
	tests := []struct {
		name string
		v    interface{}
		want map[string][]int
	}{
		{
			name: "empty",
			v:    struct{}{},
			want: map[string][]int{},
		},
		{
			name: "unexported",
			v: struct {
				unexported string
			}{},
			want: map[string][]int{},
		},
		{
			name: "one",
			v: struct {
				One string
			}{},
			want: map[string][]int{"one": {0}},
		},
		{
			name: "multiple",
			v: struct {
				ID          string
				Name        string
				Code        string
				IsActive    bool
				Theme       NestedTheme       `db:"theme,json"`
				Alternative NestedTheme       `db:"alternative,other,json"`
				Map         map[string]string `db:"jm"`
				CreatedAt   time.Time
				ModifiedAt  time.Time
				Ignored     string `db:"-"`
				Pointer     *string
				Embed
			}{},
			want: map[string][]int{
				"id":          {0},
				"name":        {1},
				"code":        {2},
				"is_active":   {3},
				"theme":       {4},
				"alternative": {5},
				"jm":          {6},
				"created_at":  {7},
				"modified_at": {8},
				"pointer":     {10},
				"play":        {11, 0},
			},
		},
		{
			name: "no_json_tag_option",
			v: struct {
				ID          string
				Alternative NestedTheme `db:"renamed,nope"`
			}{},
			want: map[string][]int{
				"id":           {0},
				"renamed":      {1},
				"renamed.id":   {1, 0},
				"renamed.name": {1, 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structType := reflect.Indirect(reflect.ValueOf(tt.v)).Type()
			if got := GetColumnToFieldIndexMap(structType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetColumnToFieldIndexMap() = %v, want %v", got, tt.want)
			}

			// Run again to check cached values:
			if got := GetColumnToFieldIndexMap(structType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetColumnToFieldIndexMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWildcardCache(t *testing.T) {
	old := columnToField
	t.Cleanup(func() {
		columnToField = old // Restore default caching.
	})

	const maxCached = 3
	columnToField = newLRU(maxCached)

	mocks := []interface{}{
		struct {
			Automatic string
			Tagged    string `db:"tagged"`
			OneTwo    string // OneTwo should be one_two in the database.
			CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
			Ignored   string `db:"-"`
		}{},
		struct {
			Number int
		}{},
		struct {
			A string
			B string
			C string
		}{},
		struct {
			Name string
		}{},
		struct {
			Name string
			Age  int
		}{},
	}
	for _, m := range mocks {
		structType := reflect.Indirect(reflect.ValueOf(m)).Type()
		orig := GetColumnToFieldIndexMap(structType)
		cached := GetColumnToFieldIndexMap(structType)

		if !reflect.DeepEqual(orig, cached) {
			t.Errorf("expected cached value %q to be equal to original %q and not empty", cached, orig)
		}
		if columnToField.l.Len() != len(columnToField.m) {
			t.Error("cache doubly linked list and map length should match")
		}
		if len(columnToField.m) > maxCached {
			t.Errorf("cache should contain %d once full, got %d instead", maxCached, len(columnToField.m))
		}
	}
}
