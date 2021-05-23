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
