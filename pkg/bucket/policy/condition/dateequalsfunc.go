package condition

import (
	"fmt"
	"net/http"
	"reflect"
	"time"
)

func toDateEqualsFuncString(n name, key Key, value time.Time) string {
	return fmt.Sprintf("%v:%v:%v", n, key, value.Format(time.RFC3339))
}

type dateEqualsFunc struct {
	k     Key
	value time.Time
}

func (f dateEqualsFunc) evaluate(values map[string][]string) bool {
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

	return f.value.Equal(t)
}

func (f dateEqualsFunc) key() Key {
	return f.k
}

func (f dateEqualsFunc) name() name {
	return dateEquals
}

func (f dateEqualsFunc) String() string {
	return toDateEqualsFuncString(dateEquals, f.k, f.value)
}

func (f dateEqualsFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	values.Add(NewStringValue(f.value.Format(time.RFC3339)))

	return map[Key]ValueSet{
		f.k: values,
	}
}

type dateNotEqualsFunc struct {
	dateEqualsFunc
}

func (f dateNotEqualsFunc) evaluate(values map[string][]string) bool {
	return !f.dateEqualsFunc.evaluate(values)
}

func (f dateNotEqualsFunc) name() name {
	return dateNotEquals
}

func (f dateNotEqualsFunc) String() string {
	return toDateEqualsFuncString(dateNotEquals, f.dateEqualsFunc.k, f.dateEqualsFunc.value)
}

func valueToTime(n name, values ValueSet) (v time.Time, err error) {
	if len(values) != 1 {
		return v, fmt.Errorf("only one value is allowed for %s condition", n)
	}

	for vs := range values {
		switch vs.GetType() {
		case reflect.String:
			s, err := vs.GetString()
			if err != nil {
				return v, err
			}
			if v, err = time.Parse(time.RFC3339, s); err != nil {
				return v, fmt.Errorf("value %s must be a time.Time string for %s condition: %w", vs, n, err)
			}
		default:
			return v, fmt.Errorf("value %s must be a time.Time for %s condition", vs, n)
		}
	}

	return v, nil

}

func newDateEqualsFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToTime(dateEquals, values)
	if err != nil {
		return nil, err
	}

	return NewDateEqualsFunc(key, v)
}

func NewDateEqualsFunc(key Key, value time.Time) (Function, error) {
	return &dateEqualsFunc{key, value}, nil
}

func newDateNotEqualsFunc(key Key, values ValueSet) (Function, error) {
	v, err := valueToTime(dateNotEquals, values)
	if err != nil {
		return nil, err
	}

	return NewDateNotEqualsFunc(key, v)
}

func NewDateNotEqualsFunc(key Key, value time.Time) (Function, error) {
	return &dateNotEqualsFunc{dateEqualsFunc{key, value}}, nil
}
