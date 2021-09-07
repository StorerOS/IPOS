package condition

import (
	"fmt"
	"net/http"
	"strconv"
)

func toNumericLessThanFuncString(n name, key Key, value int) string {
	return fmt.Sprintf("%v:%v:%v", n, key, value)
}

type numericLessThanFunc struct {
	k     Key
	value int
}

func (f numericLessThanFunc) evaluate(values map[string][]string) bool {
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

	return rvInt < f.value
}

func (f numericLessThanFunc) key() Key {
	return f.k
}

func (f numericLessThanFunc) name() name {
	return numericLessThan
}

func (f numericLessThanFunc) String() string {
	return toNumericLessThanFuncString(numericLessThan, f.k, f.value)
}

func (f numericLessThanFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	values.Add(NewIntValue(f.value))

	return map[Key]ValueSet{
		f.k: values,
	}
}

type numericLessThanEqualsFunc struct {
	numericLessThanFunc
}

func (f numericLessThanEqualsFunc) evaluate(values map[string][]string) bool {
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

	return rvInt <= f.value
}

func (f numericLessThanEqualsFunc) name() name {
	return numericLessThanEquals
}

func (f numericLessThanEqualsFunc) String() string {
	return toNumericLessThanFuncString(numericLessThanEquals, f.numericLessThanFunc.k, f.numericLessThanFunc.value)
}

func newNumericLessThanFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToInt(numericLessThan, values)
	if err != nil {
		return nil, err
	}

	return NewNumericLessThanFunc(key, v)
}

func NewNumericLessThanFunc(key Key, value int) (Function, error) {
	return &numericLessThanFunc{key, value}, nil
}

func newNumericLessThanEqualsFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToInt(numericLessThanEquals, values)
	if err != nil {
		return nil, err
	}

	return NewNumericLessThanEqualsFunc(key, v)
}

func NewNumericLessThanEqualsFunc(key Key, value int) (Function, error) {
	return &numericLessThanEqualsFunc{numericLessThanFunc{key, value}}, nil
}
