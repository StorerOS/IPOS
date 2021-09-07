package condition

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func path2BucketAndObject(path string) (bucket, object string) {
	path = strings.TrimPrefix(path, "/")
	pathComponents := strings.SplitN(path, "/", 2)

	switch len(pathComponents) {
	case 1:
		bucket = pathComponents[0]
	case 2:
		bucket = pathComponents[0]
		object = pathComponents[1]
	}
	return bucket, object
}

type Value struct {
	t reflect.Kind
	s string
	i int
	b bool
}

func (v Value) GetBool() (bool, error) {
	var err error

	if v.t != reflect.Bool {
		err = fmt.Errorf("not a bool Value")
	}

	return v.b, err
}

func (v Value) GetInt() (int, error) {
	var err error

	if v.t != reflect.Int {
		err = fmt.Errorf("not a int Value")
	}

	return v.i, err
}

func (v Value) GetString() (string, error) {
	var err error

	if v.t != reflect.String {
		err = fmt.Errorf("not a string Value")
	}

	return v.s, err
}

func (v Value) GetType() reflect.Kind {
	return v.t
}

func (v Value) MarshalJSON() ([]byte, error) {
	switch v.t {
	case reflect.String:
		return json.Marshal(v.s)
	case reflect.Int:
		return json.Marshal(v.i)
	case reflect.Bool:
		return json.Marshal(v.b)
	}

	return nil, fmt.Errorf("unknown value kind %v", v.t)
}

func (v *Value) StoreBool(b bool) {
	*v = Value{t: reflect.Bool, b: b}
}

func (v *Value) StoreInt(i int) {
	*v = Value{t: reflect.Int, i: i}
}

func (v *Value) StoreString(s string) {
	*v = Value{t: reflect.String, s: s}
}

func (v Value) String() string {
	switch v.t {
	case reflect.String:
		return v.s
	case reflect.Int:
		return strconv.Itoa(v.i)
	case reflect.Bool:
		return strconv.FormatBool(v.b)
	}

	return ""
}

func (v *Value) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		v.StoreBool(b)
		return nil
	}

	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		v.StoreInt(i)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v.StoreString(s)
		return nil
	}

	return fmt.Errorf("unknown json data '%v'", data)
}

func NewBoolValue(b bool) Value {
	value := &Value{}
	value.StoreBool(b)
	return *value
}

func NewIntValue(i int) Value {
	value := &Value{}
	value.StoreInt(i)
	return *value
}

func NewStringValue(s string) Value {
	value := &Value{}
	value.StoreString(s)
	return *value
}
