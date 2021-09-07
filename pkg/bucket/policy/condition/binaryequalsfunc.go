package condition

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"

	"github.com/storeros/ipos/pkg/s3utils"
	"github.com/storeros/ipos/pkg/set"
)

func toBinaryEqualsFuncString(n name, key Key, values set.StringSet) string {
	valueStrings := values.ToSlice()
	sort.Strings(valueStrings)

	return fmt.Sprintf("%v:%v:%v", n, key, valueStrings)
}

type binaryEqualsFunc struct {
	k      Key
	values set.StringSet
}

func (f binaryEqualsFunc) evaluate(values map[string][]string) bool {
	requestValue, ok := values[http.CanonicalHeaderKey(f.k.Name())]
	if !ok {
		requestValue = values[f.k.Name()]
	}

	fvalues := f.values.ApplyFunc(substFuncFromValues(values))
	return !fvalues.Intersection(set.CreateStringSet(requestValue...)).IsEmpty()
}

func (f binaryEqualsFunc) key() Key {
	return f.k
}

func (f binaryEqualsFunc) name() name {
	return binaryEquals
}

func (f binaryEqualsFunc) String() string {
	return toBinaryEqualsFuncString(binaryEquals, f.k, f.values)
}

func (f binaryEqualsFunc) toMap() map[Key]ValueSet {
	if !f.k.IsValid() {
		return nil
	}

	values := NewValueSet()
	for _, value := range f.values.ToSlice() {
		values.Add(NewStringValue(base64.StdEncoding.EncodeToString([]byte(value))))
	}

	return map[Key]ValueSet{
		f.k: values,
	}
}

func validateBinaryEqualsValues(n name, key Key, values set.StringSet) error {
	vslice := values.ToSlice()
	for _, s := range vslice {
		sbytes, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return err
		}
		values.Remove(s)
		s = string(sbytes)
		switch key {
		case S3XAmzCopySource:
			bucket, object := path2BucketAndObject(s)
			if object == "" {
				return fmt.Errorf("invalid value '%v' for '%v' for %v condition", s, S3XAmzCopySource, n)
			}
			if err = s3utils.CheckValidBucketName(bucket); err != nil {
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
		values.Add(s)
	}

	return nil
}

func newBinaryEqualsFunc(key Key, values ValueSet) (Function, error) {
	valueStrings, err := valuesToStringSlice(binaryEquals, values)
	if err != nil {
		return nil, err
	}

	return NewBinaryEqualsFunc(key, valueStrings...)
}

func NewBinaryEqualsFunc(key Key, values ...string) (Function, error) {
	sset := set.CreateStringSet(values...)
	if err := validateBinaryEqualsValues(binaryEquals, key, sset); err != nil {
		return nil, err
	}

	return &binaryEqualsFunc{key, sset}, nil
}
