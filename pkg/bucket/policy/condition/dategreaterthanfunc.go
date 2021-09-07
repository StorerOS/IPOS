package condition

import (
	"fmt"
	"net/http"
	"time"
)

func toDateGreaterThanFuncString(n name, key Key, value time.Time) string {
	return fmt.Sprintf("%v:%v:%v", n, key, value.Format(time.RFC3339))
}

type dateGreaterThanFunc struct {
	k     Key
	value time.Time
}

func (f dateGreaterThanFunc) evaluate(values map[string][]string) bool {
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

	return t.After(f.value)
}

func (f dateGreaterThanFunc) key() Key {
	return f.k
}

func (f dateGreaterThanFunc) name() name {
	return dateGreaterThan
}

func (f dateGreaterThanFunc) String() string {
	return toDateGreaterThanFuncString(dateGreaterThan, f.k, f.value)
}

func (f dateGreaterThanFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	values.Add(NewStringValue(f.value.Format(time.RFC3339)))

	return map[Key]ValueSet{
		f.k: values,
	}
}

type dateGreaterThanEqualsFunc struct {
	dateGreaterThanFunc
}

func (f dateGreaterThanEqualsFunc) evaluate(values map[string][]string) bool {
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

	return t.After(f.value) || t.Equal(f.value)
}

func (f dateGreaterThanEqualsFunc) name() name {
	return dateGreaterThanEquals
}

func (f dateGreaterThanEqualsFunc) String() string {
	return toDateGreaterThanFuncString(dateNotEquals, f.dateGreaterThanFunc.k, f.dateGreaterThanFunc.value)
}

func newDateGreaterThanFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToTime(dateGreaterThan, values)
	if err != nil {
		return nil, err
	}

	return NewDateGreaterThanFunc(key, v)
}

func NewDateGreaterThanFunc(key Key, value time.Time) (Function, error) {
	return &dateGreaterThanFunc{key, value}, nil
}

func newDateGreaterThanEqualsFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToTime(dateNotEquals, values)
	if err != nil {
		return nil, err
	}

	return NewDateGreaterThanEqualsFunc(key, v)
}

func NewDateGreaterThanEqualsFunc(key Key, value time.Time) (Function, error) {
	return &dateGreaterThanEqualsFunc{dateGreaterThanFunc{key, value}}, nil
}
