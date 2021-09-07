package policy

import (
	"encoding/json"
)

type Effect string

const (
	Allow Effect = "Allow"

	Deny = "Deny"
)

func (effect Effect) IsAllowed(b bool) bool {
	if effect == Allow {
		return b
	}

	return !b
}

func (effect Effect) IsValid() bool {
	switch effect {
	case Allow, Deny:
		return true
	}

	return false
}

func (effect Effect) MarshalJSON() ([]byte, error) {
	if !effect.IsValid() {
		return nil, Errorf("invalid effect '%v'", effect)
	}

	return json.Marshal(string(effect))
}

func (effect *Effect) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	e := Effect(s)
	if !e.IsValid() {
		return Errorf("invalid effect '%v'", s)
	}

	*effect = e

	return nil
}
