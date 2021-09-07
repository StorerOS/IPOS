package policy

import (
	"encoding/json"
	"io"
)

const DefaultVersion = "2012-10-17"

type Args struct {
	AccountName     string              `json:"account"`
	Action          Action              `json:"action"`
	BucketName      string              `json:"bucket"`
	ConditionValues map[string][]string `json:"conditions"`
	IsOwner         bool                `json:"owner"`
	ObjectName      string              `json:"object"`
}

type Policy struct {
	ID         ID `json:"ID,omitempty"`
	Version    string
	Statements []Statement `json:"Statement"`
}

func (policy Policy) IsAllowed(args Args) bool {
	for _, statement := range policy.Statements {
		if statement.Effect == Deny {
			if !statement.IsAllowed(args) {
				return false
			}
		}
	}

	if args.IsOwner {
		return true
	}

	for _, statement := range policy.Statements {
		if statement.Effect == Allow {
			if statement.IsAllowed(args) {
				return true
			}
		}
	}

	return false
}

func (policy Policy) IsEmpty() bool {
	return len(policy.Statements) == 0
}

func (policy Policy) isValid() error {
	if policy.Version != DefaultVersion && policy.Version != "" {
		return Errorf("invalid version '%v'", policy.Version)
	}

	for _, statement := range policy.Statements {
		if err := statement.isValid(); err != nil {
			return err
		}
	}

	return nil
}

func (policy Policy) MarshalJSON() ([]byte, error) {
	if err := policy.isValid(); err != nil {
		return nil, err
	}

	type subPolicy Policy
	return json.Marshal(subPolicy(policy))
}

func (policy *Policy) dropDuplicateStatements() {
redo:
	for i := range policy.Statements {
		for j, statement := range policy.Statements[i+1:] {
			if policy.Statements[i].Effect != statement.Effect {
				continue
			}

			if !policy.Statements[i].Principal.Equals(statement.Principal) {
				continue
			}

			if !policy.Statements[i].Actions.Equals(statement.Actions) {
				continue
			}

			if !policy.Statements[i].Resources.Equals(statement.Resources) {
				continue
			}

			if policy.Statements[i].Conditions.String() != statement.Conditions.String() {
				continue
			}
			policy.Statements = append(policy.Statements[:j], policy.Statements[j+1:]...)
			goto redo
		}
	}
}

func (policy *Policy) UnmarshalJSON(data []byte) error {
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

	*policy = p

	return nil
}

func (policy Policy) Validate(bucketName string) error {
	if err := policy.isValid(); err != nil {
		return err
	}

	for _, statement := range policy.Statements {
		if err := statement.Validate(bucketName); err != nil {
			return err
		}
	}

	return nil
}

func ParseConfig(reader io.Reader, bucketName string) (*Policy, error) {
	var policy Policy

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return nil, Errorf("%w", err)
	}

	err := policy.Validate(bucketName)
	return &policy, err
}
