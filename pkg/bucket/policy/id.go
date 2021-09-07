package policy

import (
	"encoding/json"
	"unicode/utf8"
)

type ID string

func (id ID) IsValid() bool {
	return utf8.ValidString(string(id))
}

func (id ID) MarshalJSON() ([]byte, error) {
	if !id.IsValid() {
		return nil, Errorf("invalid ID %v", id)
	}

	return json.Marshal(string(id))
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	i := ID(s)
	if !i.IsValid() {
		return Errorf("invalid ID %v", s)
	}

	*id = i

	return nil
}
