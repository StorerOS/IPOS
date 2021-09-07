package condition

import (
	"fmt"
	"net/http"
	"strconv"
)

func toNumericGreaterThanFuncString(n name, key Key, value int) string {
	return fmt.Sprintf("%v:%v:%v", n, key, value)
}

type numericGreaterThanFunc struct {
	k     Key
	value int
}

func (f numericGreaterThanFunc) evaluate(values map[string][]string) bool {
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

	return rvInt > f.value
}

func (f numericGreaterThanFunc) key() Key {
	return f.k
}

func (f numericGreaterThanFunc) name() name {
	return numericGreaterThan
}

func (f numericGreaterThanFunc) String() string {
	return toNumericGreaterThanFuncString(numericGreaterThan, f.k, f.value)
}

func (f numericGreaterThanFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	values.Add(NewIntValue(f.value))

	return map[Key]ValueSet{
		f.k: values,
	}
}

type numericGreaterThanEqualsFunc struct {
	numericGreaterThanFunc
}

func (f numericGreaterThanEqualsFunc) evaluate(values map[string][]string) bool {
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

	return rvInt >= f.value
}

func (f numericGreaterThanEqualsFunc) name() name {
	return numericGreaterThanEquals
}

func (f numericGreaterThanEqualsFunc) String() string {
	return toNumericGreaterThanFuncString(numericGreaterThanEquals, f.numericGreaterThanFunc.k, f.numericGreaterThanFunc.value)
}

func newNumericGreaterThanFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToInt(numericGreaterThan, values)
	if err != nil {
		return nil, err
	}

	return NewNumericGreaterThanFunc(key, v)
}

func NewNumericGreaterThanFunc(key Key, value int) (Function, error) {
	return &numericGreaterThanFunc{key, value}, nil
}

func newNumericGreaterThanEqualsFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToInt(numericGreaterThanEquals, values)
	if err != nil {
		return nil, err
	}

	return NewNumericGreaterThanEqualsFunc(key, v)
}

func NewNumericGreaterThanEqualsFunc(key Key, value int) (Function, error) {
	return &numericGreaterThanEqualsFunc{numericGreaterThanFunc{key, value}}, nil
}
