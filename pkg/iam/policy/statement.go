package iampolicy

import (
	"encoding/json"
	"strings"

	"github.com/storeros/ipos/pkg/bucket/policy"
	"github.com/storeros/ipos/pkg/bucket/policy/condition"
)

type Statement struct {
	SID        policy.ID           `json:"Sid,omitempty"`
	Effect     policy.Effect       `json:"Effect"`
	Actions    ActionSet           `json:"Action"`
	Resources  ResourceSet         `json:"Resource,omitempty"`
	Conditions condition.Functions `json:"Condition,omitempty"`
}

func (statement Statement) IsAllowed(args Args) bool {
	check := func() bool {
		if !statement.Actions.Match(args.Action) {
			return false
		}

		resource := args.BucketName
		if args.ObjectName != "" {
			if !strings.HasPrefix(args.ObjectName, "/") {
				resource += "/"
			}

			resource += args.ObjectName
		} else {
			resource += "/"
		}

		if !statement.Resources.Match(resource, args.ConditionValues) && !statement.isAdmin() {
			return false
		}

		return statement.Conditions.Evaluate(args.ConditionValues)
	}

	return statement.Effect.IsAllowed(check())
}
func (statement Statement) isAdmin() bool {
	for action := range statement.Actions {
		if !AdminAction(action).IsValid() {
			return false
		}
	}
	return true
}

func (statement Statement) isValid() error {
	if !statement.Effect.IsValid() {
		return Errorf("invalid Effect %v", statement.Effect)
	}

	if len(statement.Actions) == 0 {
		return Errorf("Action must not be empty")
	}

	if statement.isAdmin() {
		for action := range statement.Actions {
			keys := statement.Conditions.Keys()
			keyDiff := keys.Difference(adminActionConditionKeyMap[action])
			if !keyDiff.IsEmpty() {
				return Errorf("unsupported condition keys '%v' used for action '%v'", keyDiff, action)
			}
		}
		return nil
	}

	if len(statement.Resources) == 0 {
		return Errorf("Resource must not be empty")
	}

	if err := statement.Resources.Validate(); err != nil {
		return err
	}

	for action := range statement.Actions {
		if !statement.Resources.objectResourceExists() && !statement.Resources.bucketResourceExists() {
			return Errorf("unsupported Resource found %v for action %v", statement.Resources, action)
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

func (statement Statement) Validate() error {
	return statement.isValid()
}

func NewStatement(effect policy.Effect, actionSet ActionSet, resourceSet ResourceSet, conditions condition.Functions) Statement {
	return Statement{
		Effect:     effect,
		Actions:    actionSet,
		Resources:  resourceSet,
		Conditions: conditions,
	}
}
