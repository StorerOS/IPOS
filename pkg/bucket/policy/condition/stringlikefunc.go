package condition

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/storeros/ipos/pkg/s3utils"
	"github.com/storeros/ipos/pkg/set"
	"github.com/storeros/ipos/pkg/wildcard"
)

func toStringLikeFuncString(n name, key Key, values set.StringSet) string {
	valueStrings := values.ToSlice()
	sort.Strings(valueStrings)

	return fmt.Sprintf("%v:%v:%v", n, key, valueStrings)
}

type stringLikeFunc struct {
	k      Key
	values set.StringSet
}

func (f stringLikeFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	fvalues := f.values.ApplyFunc(substFuncFromValues(values))

	for _, v := range requestValue {
		if !fvalues.FuncMatch(wildcard.Match, v).IsEmpty() {
			return true
		}
	}

	return false
}

func (f stringLikeFunc) key() Key {
	return f.k
}

func (f stringLikeFunc) name() name {
	return stringLike
}

func (f stringLikeFunc) String() string {
	return toStringLikeFuncString(stringLike, f.k, f.values)
}

func (f stringLikeFunc) toMap() map[Key]ValueSet {
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

type stringNotLikeFunc struct {
	stringLikeFunc
}

func (f stringNotLikeFunc) evaluate(values map[string][]string) bool {
	return !f.stringLikeFunc.evaluate(values)
}

func (f stringNotLikeFunc) name() name {
	return stringNotLike
}

func (f stringNotLikeFunc) String() string {
	return toStringLikeFuncString(stringNotLike, f.stringLikeFunc.k, f.stringLikeFunc.values)
}

func validateStringLikeValues(n name, key Key, values set.StringSet) error {
	for _, s := range values.ToSlice() {
		switch key {
		case S3XAmzCopySource:
			bucket, object := path2BucketAndObject(s)
			if object == "" {
				return fmt.Errorf("invalid value '%v' for '%v' for %v condition", s, S3XAmzCopySource, n)
			}
			if err := s3utils.CheckValidBucketName(bucket); err != nil {
				return err
			}
		}
	}

	return nil
}

func newStringLikeFunc(key Key, values ValueSet) (Function, error) {
	valueStrings, err := valuesToStringSlice(stringLike, values)
	if err != nil {
		return nil, err
	}

	return NewStringLikeFunc(key, valueStrings...)
}

func NewStringLikeFunc(key Key, values ...string) (Function, error) {
	sset := set.CreateStringSet(values...)
	if err := validateStringLikeValues(stringLike, key, sset); err != nil {
		return nil, err
	}

	return &stringLikeFunc{key, sset}, nil
}

func newStringNotLikeFunc(key Key, values ValueSet) (Function, error) {
	valueStrings, err := valuesToStringSlice(stringNotLike, values)
	if err != nil {
		return nil, err
	}

	return NewStringNotLikeFunc(key, valueStrings...)
}

func NewStringNotLikeFunc(key Key, values ...string) (Function, error) {
	sset := set.CreateStringSet(values...)
	if err := validateStringLikeValues(stringNotLike, key, sset); err != nil {
		return nil, err
	}

	return &stringNotLikeFunc{stringLikeFunc{key, sset}}, nil
}
