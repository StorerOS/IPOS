package policy

import (
	"encoding/json"
	"strings"

	"github.com/storeros/ipos/pkg/bucket/policy/condition"
)

type Statement struct {
	SID        ID                  `json:"Sid,omitempty"`
	Effect     Effect              `json:"Effect"`
	Principal  Principal           `json:"Principal"`
	Actions    ActionSet           `json:"Action"`
	Resources  ResourceSet         `json:"Resource"`
	Conditions condition.Functions `json:"Condition,omitempty"`
}

func (statement Statement) IsAllowed(args Args) bool {
	check := func() bool {
		if !statement.Principal.Match(args.AccountName) {
			return false
		}

		if !statement.Actions.Contains(args.Action) {
			return false
		}

		resource := args.BucketName
		if args.ObjectName != "" {
			if !strings.HasPrefix(args.ObjectName, "/") {
				resource += "/"
			}

			resource += args.ObjectName
		}

		if !statement.Resources.Match(resource, args.ConditionValues) {
			return false
		}

		return statement.Conditions.Evaluate(args.ConditionValues)
	}

	return statement.Effect.IsAllowed(check())
}

func (statement Statement) isValid() error {
	if !statement.Effect.IsValid() {
		return Errorf("invalid Effect %v", statement.Effect)
	}

	if !statement.Principal.IsValid() {
		return Errorf("invalid Principal %v", statement.Principal)
	}

	if len(statement.Actions) == 0 {
		return Errorf("Action must not be empty")
	}

	if len(statement.Resources) == 0 {
		return Errorf("Resource must not be empty")
	}

	for action := range statement.Actions {
		if action.isObjectAction() {
			if !statement.Resources.objectResourceExists() {
				return Errorf("unsupported Resource found %v for action %v", statement.Resources, action)
			}
		} else {
			if !statement.Resources.bucketResourceExists() {
				return Errorf("unsupported Resource found %v for action %v", statement.Resources, action)
			}
		}

		keys := statement.Conditions.Keys()
		keyDiff := keys.Difference(actionConditionKeyMap[action])
		if !keyDiff.IsEmpty() {
			return Errorf("unsupported condition keys '%v' used for action '%v'", keyDiff, action)
		}
	}

	return nil
}

func (statement Statement) MarshalJSON() ([]byte, error) {
	if err := statement.isValid(); err != nil {
		return nil, err
	}

	type subStatement Statement
	ss := subStatement(statement)
	return json.Marshal(ss)
}

func (statement *Statement) UnmarshalJSON(data []byte) error {
	type subStatement Statement
	var ss subStatement

	if err := json.Unmarshal(data, &ss); err != nil {
		return err
	}

	s := Statement(ss)
	if err := s.isValid(); err != nil {
		return err
	}

	*statement = s

	return nil
}

func (statement Statement) Validate(bucketName string) error {
	if err := statement.isValid(); err != nil {
		return err
	}

	return statement.Resources.Validate(bucketName)
}

func NewStatement(effect Effect, principal Principal, actionSet ActionSet, resourceSet ResourceSet, conditions condition.Functions) Statement {
	return Statement{
		Effect:     effect,
		Principal:  principal,
		Actions:    actionSet,
		Resources:  resourceSet,
		Conditions: conditions,
	}
}
