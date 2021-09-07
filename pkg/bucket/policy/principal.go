package policy

import (
	"encoding/json"

	"github.com/storeros/ipos/pkg/set"
	"github.com/storeros/ipos/pkg/wildcard"
)

type Principal struct {
	AWS set.StringSet
}

func (p Principal) IsValid() bool {
	return len(p.AWS) != 0
}

func (p Principal) Equals(pp Principal) bool {
	return p.AWS.Equals(pp.AWS)
}

func (p Principal) Intersection(principal Principal) set.StringSet {
	return p.AWS.Intersection(principal.AWS)
}

func (p Principal) MarshalJSON() ([]byte, error) {
	if !p.IsValid() {
		return nil, Errorf("invalid principal %v", p)
	}

	type subPrincipal Principal
	sp := subPrincipal(p)
	return json.Marshal(sp)
}

func (p Principal) Match(principal string) bool {
	for _, pattern := range p.AWS.ToSlice() {
		if wildcard.MatchSimple(pattern, principal) {
			return true
		}
	}

	return false
}

func (p *Principal) UnmarshalJSON(data []byte) error {
	type subPrincipal Principal
	var sp subPrincipal

	if err := json.Unmarshal(data, &sp); err != nil {
		var s string
		if err = json.Unmarshal(data, &s); err != nil {
			return err
		}

		if s != "*" {
			return Errorf("invalid principal '%v'", s)
		}

		sp.AWS = set.CreateStringSet("*")
	}

	*p = Principal(sp)

	return nil
}

func NewPrincipal(principals ...string) Principal {
	return Principal{AWS: set.CreateStringSet(principals...)}
}
