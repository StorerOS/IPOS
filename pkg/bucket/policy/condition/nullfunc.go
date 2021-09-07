package condition

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

type nullFunc struct {
	k     Key
	value bool
}

func (f nullFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	if f.value {
		return len(requestValue) == 0
	}

	return len(requestValue) != 0
}

func (f nullFunc) key() Key {
	return f.k
}

func (f nullFunc) name() name {
	return null
}

func (f nullFunc) String() string {
	return fmt.Sprintf("%v:%v:%v", null, f.k, f.value)
}

func (f nullFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	return map[Key]ValueSet{
		f.k: NewValueSet(NewBoolValue(f.value)),
	}
}

func newNullFunc(key Key, values ValueSet) (Function, error) {
	if len(values) != 1 {
		return nil, fmt.Errorf("only one value is allowed for Null condition")
	}

	var value bool
	for v := range values {
		switch v.GetType() {
		case reflect.Bool:
			value, _ = v.GetBool()
		case reflect.String:
			var err error
			s, _ := v.GetString()
			if value, err = strconv.ParseBool(s); err != nil {
				return nil, fmt.Errorf("value must be a boolean string for Null condition")
			}
		default:
			return nil, fmt.Errorf("value must be a boolean for Null condition")
		}
	}

	return &nullFunc{key, value}, nil
}

func NewNullFunc(key Key, value bool) (Function, error) {
	return &nullFunc{key, value}, nil
}
