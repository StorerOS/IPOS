package iampolicy

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/storeros/ipos/pkg/bucket/policy"
)

const DefaultVersion = "2012-10-17"

type Args struct {
	AccountName     string                 `json:"account"`
	Action          Action                 `json:"action"`
	BucketName      string                 `json:"bucket"`
	ConditionValues map[string][]string    `json:"conditions"`
	IsOwner         bool                   `json:"owner"`
	ObjectName      string                 `json:"object"`
	Claims          map[string]interface{} `json:"claims"`
}

func (a Args) GetPolicies(policyClaimName string) ([]string, bool) {
	pname, ok := a.Claims[policyClaimName]
	if !ok {
		return nil, false
	}
	pnameStr, ok := pname.(string)
	if ok {
		return strings.Split(pnameStr, ","), true
	}
	pnameSlice, ok := pname.([]string)
	return pnameSlice, ok
}

type Policy struct {
	ID         policy.ID `json:"ID,omitempty"`
	Version    string
	Statements []Statement `json:"Statement"`
}

func (iamp Policy) IsAllowed(args Args) bool {
	for _, statement := range iamp.Statements {
		if statement.Effect == policy.Deny {
			if !statement.IsAllowed(args) {
				return false
			}
		}
	}

	if args.IsOwner {
		return true
	}

	for _, statement := range iamp.Statements {
		if statement.Effect == policy.Allow {
			if statement.IsAllowed(args) {
				return true
			}
		}
	}

	return false
}

func (iamp Policy) IsEmpty() bool {
	return len(iamp.Statements) == 0
}

func (iamp Policy) isValid() error {
	if iamp.Version != DefaultVersion && iamp.Version != "" {
		return Errorf("invalid version '%v'", iamp.Version)
	}

	for _, statement := range iamp.Statements {
		if err := statement.isValid(); err != nil {
			return err
		}
	}

	return nil
}

func (iamp Policy) MarshalJSON() ([]byte, error) {
	if err := iamp.isValid(); err != nil {
		return nil, err
	}

	type subPolicy Policy
	return json.Marshal(subPolicy(iamp))
}

func (iamp *Policy) dropDuplicateStatements() {
redo:
	for i := range iamp.Statements {
		for j, statement := range iamp.Statements[i+1:] {
			if iamp.Statements[i].Effect != statement.Effect {
				continue
			}

			if !iamp.Statements[i].Actions.Equals(statement.Actions) {
				continue
			}

			if !iamp.Statements[i].Resources.Equals(statement.Resources) {
				continue
			}

			if iamp.Statements[i].Conditions.String() != statement.Conditions.String() {
				continue
			}
			iamp.Statements = append(iamp.Statements[:j], iamp.Statements[j+1:]...)
			goto redo
		}
	}
}

func (iamp *Policy) UnmarshalJSON(data []byte) error {
	type subPolicy Policy
	var sp subPolicy
	if err := json.Unmarshal(data, &sp); err != nil {
		return err
	}

	p := Policy(sp)
	if err := p.isValid(); err != nil {
		return err
	}

	p.dropDuplicateStatements()

	*iamp = p

	return nil
}

func (iamp Policy) Validate() error {
	if err := iamp.isValid(); err != nil {
		return err
	}

	for _, statement := range iamp.Statements {
		if err := statement.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func ParseConfig(reader io.Reader) (*Policy, error) {
	var iamp Policy

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&iamp); err != nil {
		return nil, Errorf("%w", err)
	}

	return &iamp, iamp.Validate()
}
