package condition

import (
	"encoding/json"
	"fmt"
)

type ValueSet map[Value]struct{}

func (set ValueSet) Add(value Value) {
	set[value] = struct{}{}
}

func (set ValueSet) MarshalJSON() ([]byte, error) {
	var values []Value
	for k := range set {
		values = append(values, k)
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("invalid value set %v", set)
	}

	return json.Marshal(values)
}

func (set *ValueSet) UnmarshalJSON(data []byte) error {
	var v Value
	if err := json.Unmarshal(data, &v); err == nil {
		*set = make(ValueSet)
		set.Add(v)
		return nil
	}

	var values []Value
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}

	if len(values) < 1 {
		return fmt.Errorf("invalid value")
	}

	*set = make(ValueSet)
	for _, v = range values {
		if _, found := (*set)[v]; found {
			return fmt.Errorf("duplicate value found '%v'", v)
		}

		set.Add(v)
	}

	return nil
}

func NewValueSet(values ...Value) ValueSet {
	set := make(ValueSet)

	for _, value := range values {
		set.Add(value)
	}

	return set
}
