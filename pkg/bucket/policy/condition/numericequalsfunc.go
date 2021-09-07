package condition

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

func toNumericEqualsFuncString(n name, key Key, value int) string {
	return fmt.Sprintf("%v:%v:%v", n, key, value)
}

type numericEqualsFunc struct {
	k     Key
	value int
}

func (f numericEqualsFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	if len(requestValue) == 0 {
		return false
	}

	rvInt, err := strconv.Atoi(requestValue[0])
	if err != nil {
		return false
	}

	return f.value == rvInt
}

func (f numericEqualsFunc) key() Key {
	return f.k
}

func (f numericEqualsFunc) name() name {
	return numericEquals
}

func (f numericEqualsFunc) String() string {
	return toNumericEqualsFuncString(numericEquals, f.k, f.value)
}

func (f numericEqualsFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	values.Add(NewIntValue(f.value))

	return map[Key]ValueSet{
		f.k: values,
	}
}

type numericNotEqualsFunc struct {
	numericEqualsFunc
}

func (f numericNotEqualsFunc) evaluate(values map[string][]string) bool {
	return !f.numericEqualsFunc.evaluate(values)
}

func (f numericNotEqualsFunc) name() name {
	return numericNotEquals
}

func (f numericNotEqualsFunc) String() string {
	return toNumericEqualsFuncString(numericNotEquals, f.numericEqualsFunc.k, f.numericEqualsFunc.value)
}

func valueToInt(n name, values ValueSet) (v int, err error) {
	if len(values) != 1 {
		return -1, fmt.Errorf("only one value is allowed for %s condition", n)
	}

	for vs := range values {
		switch vs.GetType() {
		case reflect.Int:
			if v, err = vs.GetInt(); err != nil {
				return -1, err
			}
		case reflect.String:
			s, err := vs.GetString()
			if err != nil {
				return -1, err
			}
			if v, err = strconv.Atoi(s); err != nil {
				return -1, fmt.Errorf("value %s must be a int for %s condition: %w", vs, n, err)
			}
		default:
			return -1, fmt.Errorf("value %s must be a int for %s condition", vs, n)
		}
	}

	return v, nil

}

func newNumericEqualsFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToInt(numericEquals, values)
	if err != nil {
		return nil, err
	}

	return NewNumericEqualsFunc(key, v)
}

func NewNumericEqualsFunc(key Key, value int) (Function, error) {
	return &numericEqualsFunc{key, value}, nil
}

func newNumericNotEqualsFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToInt(numericNotEquals, values)
	if err != nil {
		return nil, err
	}

	return NewNumericNotEqualsFunc(key, v)
}

func NewNumericNotEqualsFunc(key Key, value int) (Function, error) {
	return &numericNotEqualsFunc{numericEqualsFunc{key, value}}, nil
}
