package dbscan

import (
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

type Lexer struct {
	delim        rune
	compileDelim DriverDelim
}

func newLexer(delim rune, compileDelim DriverDelim) Lexer {
	return Lexer{
		delim:        delim,
		compileDelim: compileDelim,
	}
}

type builder struct {
	byteBuf      *strings.Builder
	onParameter  bool
	currentIndex uint8
	indecies     map[string]uint8
	argNames     []string
}

func newBuilder() builder {
	return builder{
		byteBuf:      &strings.Builder{},
		onParameter:  false,
		currentIndex: 0,
		indecies:     map[string]uint8{},
		argNames:     []string{},
	}
}

//returns the index of the numbered placeholder value if not found it creates a new one
func (b *builder) indexOf(param string) uint8 {
	index, found := b.indecies[param]
	if found {
		return index
	}

	b.currentIndex += 1
	b.indecies[param] = b.currentIndex
	b.argNames = append(b.argNames, param)
	return b.currentIndex
}

func (b *builder) appendPart(str string, compileDelim DriverDelim) error {
	if b.onParameter {
		return compileDelim(b.byteBuf, b.indexOf(str))
	}

	_, err2 := b.byteBuf.WriteString(str)
	if err2 != nil {
		return err2
	}

	return nil
}

func (b builder) String() string {
	return b.byteBuf.String()
}

func wrapNamedError(err error) error {
	return errors.Wrap(err, "scany named")
}

//Compile is used for parsing and compiling sql queries
//returns the compiled query as the first parameter and the second parameter consists of all of the
func (l Lexer) Compile(sql string) (string, []string, error) {
	builder := newBuilder()
	start := 0
	pos := 0

	for {
		_rune, width := utf8.DecodeRuneInString(sql[pos:])

		if _rune == utf8.RuneError {
			err := builder.appendPart(sql[start:pos], l.compileDelim)
			if err != nil {
				return "", nil, wrapNamedError(err)
			}

			break
		} else if _rune == l.delim {
			err := builder.appendPart(sql[start:pos], l.compileDelim)
			if err != nil {
				return "", nil, wrapNamedError(err)
			}

			builder.onParameter = true
			start = pos + width
		} else if _rune != '_' && _rune != '.' && (_rune > 'z' || _rune < '1') {
			if builder.onParameter {
				err := builder.appendPart(sql[start:pos], l.compileDelim)
				if err != nil {
					return "", nil, wrapNamedError(err)
				}

				builder.onParameter = false
				start = pos
			}
		}

		pos += width
	}

	return builder.String(), builder.argNames, nil
}

type PreparedQuery struct {
	api         *API
	query       string
	namedParams []string
}

//Prepares named queries
//Approx. saves 2000ns/query on decently complex queries
func (api *API) PrepareNamed(query string, args ...interface{}) (*PreparedQuery, error) {
	compiledQuery, params, err := api.lexer.Compile(query)
	if err != nil {
		return nil, err
	}

	prep := &PreparedQuery{
		api:         api,
		query:       compiledQuery,
		namedParams: params,
	}

	errSb := strings.Builder{}

	for _, assertableStruct := range args {
		err := prep.assertStruct(assertableStruct)
		if err != nil {
			_, err2 := errSb.WriteString(err.Error())
			if err2 != nil {
				return nil, err2
			}
			_, err2 = errSb.WriteString("\n")
			if err2 != nil {
				return nil, err2
			}
		}
	}

	if errSb.Len() > 0 {
		return prep, errors.New(strings.Trim(errSb.String(), "\n")) 
	}

	return prep, nil
}

//assertStruct makes sure that it can find the struct fields when the named query is prepared
//reduces development time since the user of the library does not need to run each query to see if they map struct fields or not
func (pq *PreparedQuery) assertStruct(assertableStruct interface{}) error {
	st := reflect.TypeOf(assertableStruct)

	fieldIndexMap := pq.api.getColumnToFieldIndexMapV2(st)
	sBuilder := strings.Builder{}

	for _, fieldName := range pq.namedParams {
		res := fieldIndexMap.fieldIndexes[fieldName]
		if len(res) == 0 {
			_, err := sBuilder.WriteString("field '")
			if err != nil {
				return err
			}

			_, err = sBuilder.WriteString(fieldName)
			if err != nil {
				return err
			}

			_, err = sBuilder.WriteString("' was not found from '")
			if err != nil {
				return err
			}

			_, err = sBuilder.WriteString(st.Name())
			if err != nil {
				return err
			}

			_, err = sBuilder.WriteString("' struct.\n")
			if err != nil {
				return err
			}
		}
	}

	if sBuilder.Len() > 0 {
		return errors.New(sBuilder.String())
	}

	return nil
}

//GetQuery returns the array of the values behind the named params
func (pq *PreparedQuery) GetQuery(arg interface{}) (string, []interface{}, error) {
	args, err := pq.api.args(arg, pq.namedParams)
	return pq.query, args, err
}

//Maps the named args to corresponding fields in a structs and maps
func (api *API) args(arg interface{}, namedArgs []string) ([]interface{}, error) {
	t := reflect.TypeOf(arg)
	k := t.Kind()

	switch {
	case k == reflect.Map && t.Key().Kind() == reflect.String:
		{
			//map args

			args := make([]interface{}, 0, len(namedArgs))
			val := reflect.ValueOf(arg)

			mapKeys := val.MapKeys()
			keyMap := map[string]reflect.Value{}
			for _, key := range mapKeys {
				keyMap[(key.Interface()).(string)] = key
			}

			for _, key := range namedArgs {

				actualKey, found := keyMap[key]
				if !found {
					return nil, errors.New("value for key '" + key + "' not found")
				}
				ret := val.MapIndex(actualKey.Convert(val.Type().Key())).Interface()

				args = append(args, ret)
			}

			return args, nil
		}
	case k == reflect.Array || k == reflect.Slice:
		{
			//return bindArray(query, arg, m)
		}
	default:
		{
			//map struct fields

			args := make([]interface{}, 0, len(namedArgs))
			prep := reflect.ValueOf(arg).Elem()
			fieldIndexMap := api.getColumnToFieldIndexMapV2(prep.Type())

			for _, key := range namedArgs {
				val, err := fieldIndexMap.fieldValue(prep, key)
				if err != nil {
					return nil, err
				}
				args = append(args, val)
			}

			return args, nil
		}
	}

	return nil, errors.New("couldn't bind arguments")
}
