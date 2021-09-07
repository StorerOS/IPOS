package lifecycle

import (
	"encoding/xml"
	"io"
	"strings"
	"time"
)

var (
	errLifecycleTooManyRules      = Errorf("Lifecycle configuration allows a maximum of 1000 rules")
	errLifecycleNoRule            = Errorf("Lifecycle configuration should have at least one rule")
	errLifecycleOverlappingPrefix = Errorf("Lifecycle configuration has rules with overlapping prefix")
)

type Action int

const (
	NoneAction Action = iota
	DeleteAction
)

type Lifecycle struct {
	XMLName xml.Name `xml:"LifecycleConfiguration"`
	Rules   []Rule   `xml:"Rule"`
}

func (lc Lifecycle) IsEmpty() bool {
	return len(lc.Rules) == 0
}

func ParseLifecycleConfig(reader io.Reader) (*Lifecycle, error) {
	var lc Lifecycle
	if err := xml.NewDecoder(reader).Decode(&lc); err != nil {
		return nil, err
	}
	if err := lc.Validate(); err != nil {
		return nil, err
	}
	return &lc, nil
}

func (lc Lifecycle) Validate() error {
	if len(lc.Rules) > 1000 {
		return errLifecycleTooManyRules
	}
	if len(lc.Rules) == 0 {
		return errLifecycleNoRule
	}
	for _, r := range lc.Rules {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	for i := range lc.Rules {
		if i == len(lc.Rules)-1 {
			break
		}
		otherRules := lc.Rules[i+1:]
		for _, otherRule := range otherRules {
			if strings.HasPrefix(lc.Rules[i].Prefix(), otherRule.Prefix()) ||
				strings.HasPrefix(otherRule.Prefix(), lc.Rules[i].Prefix()) {
				return errLifecycleOverlappingPrefix
			}
		}
	}
	return nil
}

func (lc Lifecycle) FilterRuleActions(objName, objTags string) (Expiration, Transition) {
	if objName == "" {
		return Expiration{}, Transition{}
	}
	for _, rule := range lc.Rules {
		if rule.Status == Disabled {
			continue
		}
		tags := rule.Tags()
		if strings.HasPrefix(objName, rule.Prefix()) {
			if tags != "" {
				if strings.Contains(objTags, tags) {
					return rule.Expiration, Transition{}
				}
			} else {
				return rule.Expiration, Transition{}
			}
		}
	}
	return Expiration{}, Transition{}
}

func (lc Lifecycle) ComputeAction(objName, objTags string, modTime time.Time) Action {
	var action = NoneAction
	if modTime.IsZero() {
		return action
	}
	exp, _ := lc.FilterRuleActions(objName, objTags)
	if !exp.IsDateNull() {
		if time.Now().After(exp.Date.Time) {
			action = DeleteAction
		}
	}
	if !exp.IsDaysNull() {
		if time.Now().After(modTime.Add(time.Duration(exp.Days) * 24 * time.Hour)) {
			action = DeleteAction
		}
	}
	return action
}
