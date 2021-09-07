package condition

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

type booleanFunc struct {
	k     Key
	value string
}

func (f booleanFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	if len(requestValue) == 0 {
		return false
	}

	return f.value == requestValue[0]
}

func (f booleanFunc) key() Key {
	return f.k
}

func (f booleanFunc) name() name {
	return boolean
}

func (f booleanFunc) String() string {
	return fmt.Sprintf("%v:%v:%v", boolean, f.k, f.value)
}

func (f booleanFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	return map[Key]ValueSet{
		f.k: NewValueSet(NewStringValue(f.value)),
	}
}

func newBooleanFunc(key Key, values ValueSet) (Function, error) {
	if key != AWSSecureTransport {
		return nil, fmt.Errorf("only %v key is allowed for %v condition", AWSSecureTransport, boolean)
	}

	if len(values) != 1 {
		return nil, fmt.Errorf("only one value is allowed for boolean condition")
	}

	var value Value
	for v := range values {
		value = v
		switch v.GetType() {
		case reflect.Bool:
			if _, err := v.GetBool(); err != nil {
				return nil, err
			}
		case reflect.String:
			s, err := v.GetString()
			if err != nil {
				return nil, err
			}
			if _, err = strconv.ParseBool(s); err != nil {
				return nil, fmt.Errorf("value must be a boolean string for boolean condition")
			}
		default:
			return nil, fmt.Errorf("value must be a boolean for boolean condition")
		}
	}

	return &booleanFunc{key, value.String()}, nil
}

func NewBoolFunc(key Key, value string) (Function, error) {
	return &booleanFunc{key, value}, nil
}
