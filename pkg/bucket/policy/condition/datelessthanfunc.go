package condition

import (
	"fmt"
	"net/http"
	"time"
)

func toDateLessThanFuncString(n name, key Key, value time.Time) string {
	return fmt.Sprintf("%v:%v:%v", n, key, value.Format(time.RFC3339))
}

type dateLessThanFunc struct {
	k     Key
	value time.Time
}

func (f dateLessThanFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	if len(requestValue) == 0 {
		return false
	}

	t, err := time.Parse(time.RFC3339, requestValue[0])
	if err != nil {
		return false
	}

	return t.Before(f.value)
}

func (f dateLessThanFunc) key() Key {
	return f.k
}

func (f dateLessThanFunc) name() name {
	return dateLessThan
}

func (f dateLessThanFunc) String() string {
	return toDateLessThanFuncString(dateLessThan, f.k, f.value)
}

func (f dateLessThanFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	values.Add(NewStringValue(f.value.Format(time.RFC3339)))

	return map[Key]ValueSet{
		f.k: values,
	}
}

type dateLessThanEqualsFunc struct {
	dateLessThanFunc
}

func (f dateLessThanEqualsFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	if len(requestValue) == 0 {
		return false
	}

	t, err := time.Parse(time.RFC3339, requestValue[0])
	if err != nil {
		return false
	}

	return t.Before(f.value) || t.Equal(f.value)
}

func (f dateLessThanEqualsFunc) name() name {
	return dateLessThanEquals
}

func (f dateLessThanEqualsFunc) String() string {
	return toDateLessThanFuncString(dateNotEquals, f.dateLessThanFunc.k, f.dateLessThanFunc.value)
}

func newDateLessThanFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToTime(dateLessThan, values)
	if err != nil {
		return nil, err
	}

	return NewDateLessThanFunc(key, v)
}

func NewDateLessThanFunc(key Key, value time.Time) (Function, error) {
	return &dateLessThanFunc{key, value}, nil
}

func newDateLessThanEqualsFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToTime(dateNotEquals, values)
	if err != nil {
		return nil, err
	}

	return NewDateLessThanEqualsFunc(key, v)
}

func NewDateLessThanEqualsFunc(key Key, value time.Time) (Function, error) {
	return &dateLessThanEqualsFunc{dateLessThanFunc{key, value}}, nil
}
