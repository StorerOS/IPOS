package policy

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/storeros/ipos/pkg/set"
)

type ActionSet map[Action]struct{}

func (actionSet ActionSet) Add(action Action) {
	actionSet[action] = struct{}{}
}

func (actionSet ActionSet) Contains(action Action) bool {
	_, found := actionSet[action]
	return found
}

func (actionSet ActionSet) Equals(sactionSet ActionSet) bool {
	if len(actionSet) != len(sactionSet) {
		return false
	}

	for k := range actionSet {
		if _, ok := sactionSet[k]; !ok {
			return false
		}
	}

	return true
}

func (actionSet ActionSet) Intersection(sset ActionSet) ActionSet {
	nset := NewActionSet()
	for k := range actionSet {
		if _, ok := sset[k]; ok {
			nset.Add(k)
		}
	}

	return nset
}

func (actionSet ActionSet) MarshalJSON() ([]byte, error) {
	if len(actionSet) == 0 {
		return nil, Errorf("empty actions not allowed")
	}

	return json.Marshal(actionSet.ToSlice())
}

func (actionSet ActionSet) String() string {
	actions := []string{}
	for action := range actionSet {
		actions = append(actions, string(action))
	}
	sort.Strings(actions)

	return fmt.Sprintf("%v", actions)
}

func (actionSet ActionSet) ToSlice() []Action {
	actions := []Action{}
	for action := range actionSet {
		actions = append(actions, action)
	}

	return actions
}

func (actionSet *ActionSet) UnmarshalJSON(data []byte) error {
	var sset set.StringSet
	if err := json.Unmarshal(data, &sset); err != nil {
		return err
	}

	if len(sset) == 0 {
		return Errorf("empty actions not allowed")
	}

	*actionSet = make(ActionSet)
	for _, s := range sset.ToSlice() {
		action, err := parseAction(s)
		if err != nil {
			return err
		}

		actionSet.Add(action)
	}

	return nil
}

func NewActionSet(actions ...Action) ActionSet {
	actionSet := make(ActionSet)
	for _, action := range actions {
		actionSet.Add(action)
	}

	return actionSet
}
