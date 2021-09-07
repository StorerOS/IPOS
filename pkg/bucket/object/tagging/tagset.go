package tagging

import (
	"encoding/xml"
)

type TagSet struct {
	XMLName xml.Name `xml:"TagSet"`
	Tags    []Tag    `xml:"Tag"`
}

func (t TagSet) ContainsDuplicateTag() bool {
	x := make(map[string]struct{}, len(t.Tags))

	for _, t := range t.Tags {
		if _, has := x[t.Key]; has {
			return true
		}
		x[t.Key] = struct{}{}
	}

	return false
}
