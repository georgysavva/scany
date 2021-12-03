package dbscan

import (
	"reflect"
	"testing"
)

type testStruct struct {
	Field   string
	Field2  int
	Field3  bool
	Field4  int8
	Field5  int16
	Field6  int32
	Field7  int64
	Field8  float32
	Field9  float64
	Field12 uint
	Field14 uint8
	Field15 uint16
	Field16 uint32
	Field17 uint64
}

func prepReflect(ifc interface{}) reflect.Value {
	return reflect.ValueOf(ifc).Elem()
}

func TestMissingField(t *testing.T) {

	prepped := prepReflect(&testStruct{Field2: 123})
	imap := DefaultAPI.getColumnToFieldIndexMapV2(prepped.Type())

	_, err := imap.fieldValue(prepped, "Field22")

	if err != nil {
		if err.Error() != "field 'Field22' not found" {
			t.Error("Function errored:" + err.Error())
		}
	}
}

func TestCompleteStruct(t *testing.T) {
	l := newLexer(':', SequentialDollarDelim)

	_, argNames, err := l.Compile("SELECT :field FROM :field2")
	if err != nil {
		t.Error("ERRORED WHILE TRYING TO COMPILE QUERY: " + err.Error())
	}

	val := "TESTING 123"
	val2 := 332211

	args, err := DefaultAPI.args(&testStruct{Field: val, Field2: val2}, argNames)
	if err != nil {
		t.Error("ERRORED WHILE TRYING TO GET ARGS: " + err.Error())
	}

	if args[0].(string) != val {
		t.Fatal("Expected: '" + val + "' but got: '" + args[0].(string) + "'.")
	}

	if args[1].(int) != val2 {
		t.Fatal("Expected: '" + val + "' but got: '" + args[1].(string) + "'.")
	}
}

func TestCompleteMap(t *testing.T) {
	l := newLexer(':', SequentialDollarDelim)

	_, argNames, err := l.Compile("SELECT :test123 FROM :Test123")
	if err != nil {
		t.Error("ERRORED WHILE TRYING TO COMPILE QUERY: " + err.Error())
	}

	val := "Value1"
	val2 := "Value2"

	args, err := DefaultAPI.args(map[string]string{"test123": val, "Test123": val2}, argNames)
	if err != nil {
		t.Error("ERRORED WHILE TRYING GET ARGS: " + err.Error())
	}

	if args[0].(string) != val {
		t.Fatal("Expected: '" + val + "' but got: '" + args[0].(string) + "'.")
	}

	if args[1].(string) != val2 {
		t.Fatal("Expected: '" + val + "' but got: '" + args[1].(string) + "'.")
	}
}

func TestPrepareAssert(t *testing.T) {
	slicesEqual := func(slice []string, slice2 []string) bool {
		l := len(slice)
		if l != len(slice2) {
			return false
		}

		for i := 0; i < l; i++ {
			if slice[i] != slice2[i] {
				return false
			}
		}

		return true
	}

	api, err := NewAPI(WithLexer(':', SequentialDollarDelim))
	if err != nil {
		t.Fatal("Errored during api initialisation", err)
	}

	type Car struct {
		Model string
		Year  int
		Value float64
	}

	type testCase struct {
		input          string
		expectedParams []string
		expectedQuery  string
		expectedErrStr string
	}

	testCases := []testCase{
		{ //NORMAL
			input:          "SELECT * FROM cars WHERE model = :model",
			expectedParams: []string{"model"},
			expectedQuery:  "SELECT * FROM cars WHERE model = $1",
			expectedErrStr: "",
		},
		{ //TYPO
			input:          "SELECT * FROM cars WHERE model = :modeel",
			expectedParams: []string{"modeel"},
			expectedQuery:  "SELECT * FROM cars WHERE model = $1",
			expectedErrStr: "field 'modeel' was not found from 'Car' struct.",
		},
	}

	for _, testCase := range testCases {
		pq, err := api.PrepareNamed(testCase.input, Car{})
		if err != nil {
			if testCase.expectedErrStr == "" {
				t.Fatal("Errored while trying to prepare query", err)
			} else if testCase.expectedErrStr != err.Error() {
				t.Fatal("Expected error: '" + testCase.expectedErrStr + "', but got: '" + err.Error() + "'")
			}
		}

		if pq.query != testCase.expectedQuery {
			t.Error("Expected: '" + testCase.expectedQuery + "', but got: '" + pq.query + "'")
		}

		if !slicesEqual(pq.namedParams, testCase.expectedParams) {
			t.Errorf("Expected: %v, but got: %v", testCase.expectedParams, pq.namedParams)
		}
	}
}

type TestStruct struct {
	Name  string
	Email string
}

func TestCompile(t *testing.T) {
	l := newLexer(':', SequentialDollarDelim)

	str, _, err := l.Compile("SELECT * FROM users WHERE name = :name AND email = :email")
	if err != nil {
		t.Error("Failed: ", err.Error())
	}

	expected := "SELECT * FROM users WHERE name = $1 AND email = $2"

	if str != expected {
		t.Error("Expected \"" + expected + "\" but was \"" + str + "\"")
	}
}

func TestCompile2(t *testing.T) {
	l := newLexer(':', SequentialDollarDelim)

	str, _, err := l.Compile(":name :email :name")
	if err != nil {
		t.Error("Failed: ", err.Error())
	}

	expected := "$1 $2 $1"

	if str != expected {
		t.Error("Expected \"" + expected + "\" but was \"" + str + "\"")
	}
}

func TestCompile3(t *testing.T) {
	l := newLexer(':', SequentialDollarDelim)

	str, _, err := l.Compile(":name:email:name:email")
	if err != nil {
		t.Error("Failed: ", err.Error())
	}

	expected := "$1$2$1$2"

	if str != expected {
		t.Error("Expected \"" + expected + "\" but was \"" + str + "\"")
	}
}

func TestCompile4_Nested(t *testing.T) {

	/*
		type CreditCard struct {
			Issuer     string
			ExpireDate time.Time
		}

		type BankingInfo struct {
			CreditCard CreditCard
		}

		type User struct {
			BankingInfo BankingInfo
		}
	*/

	l := newLexer(':', SequentialDollarDelim)

	str, _, err := l.Compile(":banking_info.credit_card.issuer :banking_info.credit_card.expire_date")
	if err != nil {
		t.Error("Failed: ", err.Error())
	}

	expected := "$1 $2"

	if str != expected {
		t.Error("Expected \"" + expected + "\" but was \"" + str + "\"")
	}
}

func BenchmarkLexer(b *testing.B) {
	l := newLexer(':', SequentialDollarDelim)

	for i := 0; i < b.N; i++ {
		_, _, err := l.Compile(":name:email:name:email")
		if err != nil {
			b.Error("Failed: ", err.Error())
		}
	}
}

func BenchmarkIndexMap(b *testing.B) {
	ts := prepReflect(&testStruct{Field: "Field"})

	for i := 0; i < b.N; i++ {
		imap := DefaultAPI.getColumnToFieldIndexMapV2(ts.Type())

		_, err := imap.fieldValue(ts, "field")
		if err != nil {
			b.Error(err.Error())
		}
	}
}

func BenchmarkIndexMap_2(b *testing.B) {
	ts := prepReflect(&testStruct{Field: "Field"})

	for i := 0; i < b.N; i++ {
		imap := DefaultAPI.getColumnToFieldIndexMapV2(ts.Type())

		_, err := imap.fieldValue(ts, "field")
		if err != nil {
			b.Error(err.Error())
		}
	}
}

func BenchmarkIndexMap_3(b *testing.B) {
	ts := prepReflect(&testStruct{Field: "Field"})

	for i := 0; i < b.N; i++ {
		imap := DefaultAPI.getColumnToFieldIndexMapV2(ts.Type())

		_, err := imap.fieldValue(ts, "field")
		if err != nil {
			b.Error(err.Error())
		}

		_, err = imap.fieldValue(ts, "field2")
		if err != nil {
			b.Error(err.Error())
		}

		_, err = imap.fieldValue(ts, "field3")
		if err != nil {
			b.Error(err.Error())
		}
	}
}

func BenchmarkIndexMap_4(b *testing.B) {
	ts := prepReflect(&testStruct{Field: "Field"})

	for i := 0; i < b.N; i++ {
		imap := DefaultAPI.getColumnToFieldIndexMapV2(ts.Type())

		for x := 0; x < 5; x++ {
			_, err := imap.fieldValue(ts, "field")
			if err != nil {
				b.Error(err.Error())
			}

			_, err = imap.fieldValue(ts, "field2")
			if err != nil {
				b.Error(err.Error())
			}

			_, err = imap.fieldValue(ts, "field3")
			if err != nil {
				b.Error(err.Error())
			}
		}
	}
}

func BenchmarkReflectedMap(b *testing.B) {
	l := newLexer(':', SequentialDollarDelim)

	_, argNames, err := l.Compile("SELECT :test123 FROM :Test123")
	if err != nil {
		b.Error("ERRORED WHILE TRYING TO COMPILE QUERY: " + err.Error())
	}

	val := "Value1"
	val2 := "Value2"

	for i := 0; i < b.N; i++ {
		args, err := DefaultAPI.args(map[string]string{"test123": val, "Test123": val2}, argNames)
		if err != nil {
			b.Error("ERRORED WHILE TRYING GET ARGS: " + err.Error())
		}

		if args[0].(string) != val {
			b.Fatal("Expected: '" + val + "' but got: '" + args[0].(string) + "'.")
		}

		if args[1].(string) != val2 {
			b.Fatal("Expected: '" + val + "' but got: '" + args[1].(string) + "'.")
		}

	}

}
