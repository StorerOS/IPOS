package condition

import (
	"encoding/json"
	"fmt"
	"sort"
)

type Function interface {
	evaluate(values map[string][]string) bool

	key() Key

	name() name

	String() string

	toMap() map[Key]ValueSet
}

type Functions []Function

func (functions Functions) Evaluate(values map[string][]string) bool {
	for _, f := range functions {
		if !f.evaluate(values) {
			return false
		}
	}

	return true
}

func (functions Functions) Keys() KeySet {
	keySet := NewKeySet()

	for _, f := range functions {
		keySet.Add(f.key())
	}

	return keySet
}

func (functions Functions) MarshalJSON() ([]byte, error) {
	nm := make(map[name]map[Key]ValueSet)

	for _, f := range functions {
		if _, ok := nm[f.name()]; ok {
			for k, v := range f.toMap() {
				nm[f.name()][k] = v
			}
		} else {
			nm[f.name()] = f.toMap()
		}
	}

	return json.Marshal(nm)
}

func (functions Functions) String() string {
	funcStrings := []string{}
	for _, f := range functions {
		s := fmt.Sprintf("%v", f)
		funcStrings = append(funcStrings, s)
	}
	sort.Strings(funcStrings)

	return fmt.Sprintf("%v", funcStrings)
}

var conditionFuncMap = map[name]func(Key, ValueSet) (Function, error){
	stringEquals:              newStringEqualsFunc,
	stringNotEquals:           newStringNotEqualsFunc,
	stringEqualsIgnoreCase:    newStringEqualsIgnoreCaseFunc,
	stringNotEqualsIgnoreCase: newStringNotEqualsIgnoreCaseFunc,
	binaryEquals:              newBinaryEqualsFunc,
	stringLike:                newStringLikeFunc,
	stringNotLike:             newStringNotLikeFunc,
	ipAddress:                 newIPAddressFunc,
	notIPAddress:              newNotIPAddressFunc,
	null:                      newNullFunc,
	boolean:                   newBooleanFunc,
	numericEquals:             newNumericEqualsFunc,
	numericNotEquals:          newNumericNotEqualsFunc,
	numericLessThan:           newNumericLessThanFunc,
	numericLessThanEquals:     newNumericLessThanEqualsFunc,
	numericGreaterThan:        newNumericGreaterThanFunc,
	numericGreaterThanEquals:  newNumericGreaterThanEqualsFunc,
	dateEquals:                newDateEqualsFunc,
	dateNotEquals:             newDateNotEqualsFunc,
	dateLessThan:              newDateLessThanFunc,
	dateLessThanEquals:        newDateLessThanEqualsFunc,
	dateGreaterThan:           newDateGreaterThanFunc,
	dateGreaterThanEquals:     newDateGreaterThanEqualsFunc,
}

func (functions *Functions) UnmarshalJSON(data []byte) error {
	nm := make(map[string]map[string]ValueSet)
	if err := json.Unmarshal(data, &nm); err != nil {
		return err
	}

	if len(nm) == 0 {
		return fmt.Errorf("condition must not be empty")
	}

	funcs := []Function{}
	for nameString, args := range nm {
		n, err := parseName(nameString)
		if err != nil {
			return err
		}

		for keyString, values := range args {
			key, err := parseKey(keyString)
			if err != nil {
				return err
			}

			vfn, ok := conditionFuncMap[n]
			if !ok {
				return fmt.Errorf("condition %v is not handled", n)
			}

			f, err := vfn(key, values)
			if err != nil {
				return err
			}

			funcs = append(funcs, f)
		}
	}

	*functions = funcs

	return nil
}

func (functions Functions) GobEncode() ([]byte, error) {
	return functions.MarshalJSON()
}

func (functions *Functions) GobDecode(data []byte) error {
	return functions.UnmarshalJSON(data)
}

func NewFunctions(functions ...Function) Functions {
	return Functions(functions)
}
