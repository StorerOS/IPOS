package condition

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/storeros/ipos/pkg/s3utils"
	"github.com/storeros/ipos/pkg/set"
)

func toStringEqualsFuncString(n name, key Key, values set.StringSet) string {
	valueStrings := values.ToSlice()
	sort.Strings(valueStrings)

	return fmt.Sprintf("%v:%v:%v", n, key, valueStrings)
}

type stringEqualsFunc struct {
	k      Key
	values set.StringSet
}

func (f stringEqualsFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	fvalues := f.values.ApplyFunc(substFuncFromValues(values))
	return !fvalues.Intersection(set.CreateStringSet(requestValue...)).IsEmpty()
}

func (f stringEqualsFunc) key() Key {
	return f.k
}

func (f stringEqualsFunc) name() name {
	return stringEquals
}

func (f stringEqualsFunc) String() string {
	return toStringEqualsFuncString(stringEquals, f.k, f.values)
}

func (f stringEqualsFunc) toMap() map[Key]ValueSet {
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

type stringNotEqualsFunc struct {
	stringEqualsFunc
}

func (f stringNotEqualsFunc) evaluate(values map[string][]string) bool {
	return !f.stringEqualsFunc.evaluate(values)
}

func (f stringNotEqualsFunc) name() name {
	return stringNotEquals
}

func (f stringNotEqualsFunc) String() string {
	return toStringEqualsFuncString(stringNotEquals, f.stringEqualsFunc.k, f.stringEqualsFunc.values)
}

func valuesToStringSlice(n name, values ValueSet) ([]string, error) {
	valueStrings := []string{}

	for value := range values {
		s, err := value.GetString()
		if err != nil {
			return nil, fmt.Errorf("value must be a string for %v condition", n)
		}

		valueStrings = append(valueStrings, s)
	}

	return valueStrings, nil
}

func validateStringEqualsValues(n name, key Key, values set.StringSet) error {
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
		case S3XAmzServerSideEncryption, S3XAmzServerSideEncryptionCustomerAlgorithm:
			if s != "AES256" {
				return fmt.Errorf("invalid value '%v' for '%v' for %v condition", s, S3XAmzServerSideEncryption, n)
			}
		case S3XAmzMetadataDirective:
			if s != "COPY" && s != "REPLACE" {
				return fmt.Errorf("invalid value '%v' for '%v' for %v condition", s, S3XAmzMetadataDirective, n)
			}
		case S3XAmzContentSha256:
			if s == "" {
				return fmt.Errorf("invalid empty value for '%v' for %v condition", S3XAmzContentSha256, n)
			}
		}
	}

	return nil
}

func newStringEqualsFunc(key Key, values ValueSet) (Function, error) {
	valueStrings, err := valuesToStringSlice(stringEquals, values)
	if err != nil {
		return nil, err
	}

	return NewStringEqualsFunc(key, valueStrings...)
}

func NewStringEqualsFunc(key Key, values ...string) (Function, error) {
	sset := set.CreateStringSet(values...)
	if err := validateStringEqualsValues(stringEquals, key, sset); err != nil {
		return nil, err
	}

	return &stringEqualsFunc{key, sset}, nil
}

func newStringNotEqualsFunc(key Key, values ValueSet) (Function, error) {
	valueStrings, err := valuesToStringSlice(stringNotEquals, values)
	if err != nil {
		return nil, err
	}

	return NewStringNotEqualsFunc(key, valueStrings...)
}

func NewStringNotEqualsFunc(key Key, values ...string) (Function, error) {
	sset := set.CreateStringSet(values...)
	if err := validateStringEqualsValues(stringNotEquals, key, sset); err != nil {
		return nil, err
	}

	return &stringNotEqualsFunc{stringEqualsFunc{key, sset}}, nil
}
