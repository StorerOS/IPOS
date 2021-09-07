package lifecycle

import (
	"encoding/xml"

	"github.com/storeros/ipos/pkg/bucket/object/tagging"
)

type And struct {
	XMLName xml.Name      `xml:"And"`
	Prefix  string        `xml:"Prefix,omitempty"`
	Tags    []tagging.Tag `xml:"Tag,omitempty"`
}

var errDuplicateTagKey = Errorf("Duplicate Tag Keys are not allowed")

func (a And) isEmpty() bool {
	return len(a.Tags) == 0 && a.Prefix == ""
}

func (a And) Validate() error {
	if a.ContainsDuplicateTag() {
		return errDuplicateTagKey
	}
	for _, t := range a.Tags {
		if err := t.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (a And) ContainsDuplicateTag() bool {
	x := make(map[string]struct{}, len(a.Tags))

	for _, t := range a.Tags {
		if _, has := x[t.Key]; has {
			return true
		}
		x[t.Key] = struct{}{}
	}

	return false
}
