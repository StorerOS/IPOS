package condition

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/storeros/ipos/pkg/set"
)

func toStringEqualsIgnoreCaseFuncString(n name, key Key, values set.StringSet) string {
	valueStrings := values.ToSlice()
	sort.Strings(valueStrings)

	return fmt.Sprintf("%v:%v:%v", n, key, valueStrings)
}

type stringEqualsIgnoreCaseFunc struct {
	k      Key
	values set.StringSet
}

func (f stringEqualsIgnoreCaseFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	fvalues := f.values.ApplyFunc(substFuncFromValues(values))

	for _, v := range requestValue {
		if !fvalues.FuncMatch(strings.EqualFold, v).IsEmpty() {
			return true
		}
	}

	return false
}

func (f stringEqualsIgnoreCaseFunc) key() Key {
	return f.k
}

func (f stringEqualsIgnoreCaseFunc) name() name {
	return stringEqualsIgnoreCase
}

func (f stringEqualsIgnoreCaseFunc) String() string {
	return toStringEqualsIgnoreCaseFuncString(stringEqualsIgnoreCase, f.k, f.values)
}

func (f stringEqualsIgnoreCaseFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	for _, value := range f.values.ToSlice() {
		values.Add(NewStringValue(value))
	}

	return map[Key]ValueSet{
		f.k: values,
	}
}

type stringNotEqualsIgnoreCaseFunc struct {
	stringEqualsIgnoreCaseFunc
}

func (f stringNotEqualsIgnoreCaseFunc) evaluate(values map[string][]string) bool {
	return !f.stringEqualsIgnoreCaseFunc.evaluate(values)
}

func (f stringNotEqualsIgnoreCaseFunc) name() name {
	return stringNotEqualsIgnoreCase
}

func (f stringNotEqualsIgnoreCaseFunc) String() string {
	return toStringEqualsIgnoreCaseFuncString(stringNotEqualsIgnoreCase, f.stringEqualsIgnoreCaseFunc.k, f.stringEqualsIgnoreCaseFunc.values)
}

func validateStringEqualsIgnoreCaseValues(n name, key Key, values set.StringSet) error {
	return validateStringEqualsValues(n, key, values)
}

func newStringEqualsIgnoreCaseFunc(key Key, values ValueSet) (Function, error) {
	valueStrings, err := valuesToStringSlice(stringEqualsIgnoreCase, values)
	if err != nil {
		return nil, err
	}

	return NewStringEqualsIgnoreCaseFunc(key, valueStrings...)
}

func NewStringEqualsIgnoreCaseFunc(key Key, values ...string) (Function, error) {
	sset := set.CreateStringSet(values...)
	if err := validateStringEqualsIgnoreCaseValues(stringEqualsIgnoreCase, key, sset); err != nil {
		return nil, err
	}

	return &stringEqualsIgnoreCaseFunc{key, sset}, nil
}

func newStringNotEqualsIgnoreCaseFunc(key Key, values ValueSet) (Function, error) {
	valueStrings, err := valuesToStringSlice(stringNotEqualsIgnoreCase, values)
	if err != nil {
		return nil, err
	}

	return NewStringNotEqualsIgnoreCaseFunc(key, valueStrings...)
}

func NewStringNotEqualsIgnoreCaseFunc(key Key, values ...string) (Function, error) {
	sset := set.CreateStringSet(values...)
	if err := validateStringEqualsIgnoreCaseValues(stringNotEqualsIgnoreCase, key, sset); err != nil {
		return nil, err
	}

	return &stringNotEqualsIgnoreCaseFunc{stringEqualsIgnoreCaseFunc{key, sset}}, nil
}
