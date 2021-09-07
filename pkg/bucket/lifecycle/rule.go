package lifecycle

import (
	"bytes"
	"encoding/xml"
)

type Status string

const (
	Enabled  Status = "Enabled"
	Disabled Status = "Disabled"
)

type Rule struct {
	XMLName                     xml.Name                    `xml:"Rule"`
	ID                          string                      `xml:"ID,omitempty"`
	Status                      Status                      `xml:"Status"`
	Filter                      Filter                      `xml:"Filter,omitempty"`
	Expiration                  Expiration                  `xml:"Expiration,omitempty"`
	Transition                  Transition                  `xml:"Transition,omitempty"`
	NoncurrentVersionExpiration NoncurrentVersionExpiration `xml:"NoncurrentVersionExpiration,omitempty"`
	NoncurrentVersionTransition NoncurrentVersionTransition `xml:"NoncurrentVersionTransition,omitempty"`
}

var (
	errInvalidRuleID           = Errorf("ID must be less than 255 characters")
	errEmptyRuleStatus         = Errorf("Status should not be empty")
	errInvalidRuleStatus       = Errorf("Status must be set to either Enabled or Disabled")
	errMissingExpirationAction = Errorf("No expiration action found")
)

func (r Rule) validateID() error {
	if len(string(r.ID)) > 255 {
		return errInvalidRuleID
	}
	return nil
}

func (r Rule) validateStatus() error {
	if len(r.Status) == 0 {
		return errEmptyRuleStatus
	}

	if r.Status != Enabled && r.Status != Disabled {
		return errInvalidRuleStatus
	}
	return nil
}

func (r Rule) validateAction() error {
	if r.Expiration == (Expiration{}) {
		return errMissingExpirationAction
	}
	return nil
}

func (r Rule) validateFilter() error {
	return r.Filter.Validate()
}

func (r Rule) Prefix() string {
	if r.Filter.Prefix != "" {
		return r.Filter.Prefix
	}
	if r.Filter.And.Prefix != "" {
		return r.Filter.And.Prefix
	}
	return ""
}

func (r Rule) Tags() string {
	if !r.Filter.Tag.IsEmpty() {
		return r.Filter.Tag.String()
	}
	if len(r.Filter.And.Tags) != 0 {
		var buf bytes.Buffer
		for _, t := range r.Filter.And.Tags {
			if buf.Len() > 0 {
				buf.WriteString("&")
			}
			buf.WriteString(t.String())
		}
		return buf.String()
	}
	return ""
}

func (r Rule) Validate() error {
	if err := r.validateID(); err != nil {
		return err
	}
	if err := r.validateStatus(); err != nil {
		return err
	}
	if err := r.validateAction(); err != nil {
		return err
	}
	if err := r.validateFilter(); err != nil {
		return err
	}
	return nil
}
