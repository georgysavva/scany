package scany // nolint: testpackage

import (
	"testing"
	"time"
)

type mock struct {
	Automatic string
	Tagged    string `db:"tagged"`
	OneTwo    string // OneTwo should be one_two in the database.
	CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
	Ignored   string `db:"-"`
}

type embedMock struct {
	Before int
	mock
	After string
}

type numericMock struct {
	Number int
}

type simpleMultiEmbedMock struct {
	mock
	numericMock
}

type multiEmbedMock struct {
	A string
	mock
	B string
	numericMock
	C string
}

type emptyEmbed struct{}

type nameMock struct {
	emptyEmbed //nolint: unused
	Name       string
}

type jsonMock struct {
	ID         string
	Name       string
	Code       string
	IsActive   bool
	Theme      NestedMock `db:"theme,json"`
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type HasNestedMock struct {
	ID         string
	Name       string
	Code       string
	IsActive   bool
	Theme      NestedMock
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type HasPointerNestedMock struct {
	ID         string
	Name       string
	Code       string
	IsActive   bool
	Theme      *NestedMock
	CreatedAt  time.Time
	ModifiedAt time.Time
}

type NestedMock struct {
	PrimaryColor   string
	SecondaryColor string
	TextColor      string
	TextUppercase  bool
	SourceHeadings string
	SourceBody     string
	SourceDefault  string
}

func TestWildcard(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		v    interface{}
		desc string
		want string
	}{
		{
			v:    emptyEmbed{},
			desc: "empty",
		},
		{
			v: struct {
				unexported int
			}{},
			desc: "unexported",
			want: "",
		},
		{
			v: &struct {
				unexported int
			}{},
			desc: "unexported pointer",
			want: "",
		},
		{
			v: struct {
				One int
			}{},
			desc: "single",
			want: `"one"`,
		},
		{
			v: &struct {
				One int
			}{},
			desc: "pointer single",
			want: `"one"`,
		},
		{
			v: mock{
				Automatic: "auto string",
				Tagged:    "tag string",
			},
			desc: "mock",
			want: `"automatic","tagged","one_two","CamelCase"`,
		},
		{
			v: mock{
				Automatic: "auto string",
				Tagged:    "tag string",
			},
			desc: "cached",
			want: `"automatic","tagged","one_two","CamelCase"`,
		},
		{
			v: struct {
				Automatic string
				Tagged    string `db:"tagged"`
				OneTwo    string // OneTwo should be one_two in the database.
				CamelCase string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
				Ignored   string `db:"-"`
			}{},
			desc: "anonymous",
			want: `"automatic","tagged","one_two","CamelCase"`,
		},
		{
			v: struct {
				Automatic  string
				Tagged     string `db:"tagged"`
				OneTwo     string // OneTwo should be one_two in the database.
				CamelCase  string `db:"CamelCase"` // CamelCase should not be normalized to camel_case.
				Ignored    string `db:"-"`
				Copy       string
				Duplicated string `db:"copy"`
			}{},
			desc: "duplicated",
			want: `"automatic","tagged","one_two","CamelCase","copy"`,
		},
		{
			v:    embedMock{},
			desc: "embed",
			want: `"before","automatic","tagged","one_two","CamelCase","after"`,
		},
		{
			v:    numericMock{},
			desc: "numeric",
			want: `"number"`,
		},
		{
			v:    simpleMultiEmbedMock{},
			desc: "multisimpleembed",
			want: `"automatic","tagged","one_two","CamelCase","number"`,
		},
		{
			v:    multiEmbedMock{},
			desc: "multiembed",
			want: `"a","automatic","tagged","one_two","CamelCase","b","number","c"`,
		},
		{
			v:    &mock{},
			desc: "pointer",
			want: `"automatic","tagged","one_two","CamelCase"`,
		},
		{
			v:    &nameMock{},
			desc: "namemock",
			want: `"name"`,
		},
		{
			v:    &jsonMock{},
			desc: "json",
			want: `"id","name","code","is_active","theme","created_at","modified_at"`,
		},
		{
			v:    &HasNestedMock{},
			desc: "HasNestedMock",
			want: `"id","name","code","is_active","theme.primary_color" as "theme.primary_color","theme.secondary_color" as "theme.secondary_color","theme.text_color" as "theme.text_color","theme.text_uppercase" as "theme.text_uppercase","theme.source_headings" as "theme.source_headings","theme.source_body" as "theme.source_body","theme.source_default" as "theme.source_default","theme","created_at","modified_at"`, // nolint: lll
		},
		{
			v:    &HasPointerNestedMock{},
			desc: "HasNestedMock",
			want: `"id","name","code","is_active","theme.primary_color" as "theme.primary_color","theme.secondary_color" as "theme.secondary_color","theme.text_color" as "theme.text_color","theme.text_uppercase" as "theme.text_uppercase","theme.source_headings" as "theme.source_headings","theme.source_body" as "theme.source_body","theme.source_default" as "theme.source_default","theme","created_at","modified_at"`, // nolint: lll
		},
		{
			// Testing an edge case:
			// Regular fields containing dots are aliased even when unnecessary, and this should be okay.
			// This is a conscious design decision to reduce complexity avoiding leaking internal details from
			// internal/structref through the fields() function.
			v: struct {
				RegularFieldWithDots string `db:"regular.field.with.dots"`
			}{},
			desc: "RegularFieldWithDots",
			want: `"regular.field.with.dots" as "regular.field.with.dots"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if got := Wildcard(tc.v); tc.want != got {
				t.Errorf("expected expression to be %v, got %v instead", tc.want, got)
			}
		})
	}
}

func BenchmarkWildcardCached(b *testing.B) {
	m := mock{}
	Wildcard(m)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Wildcard(m)
	}
}
