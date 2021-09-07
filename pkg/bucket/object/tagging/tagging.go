package tagging

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/url"
)

const (
	maxTags           = 10
	maxTagKeyLength   = 128
	maxTagValueLength = 256
)

var (
	ErrTooManyTags     = Errorf("Object tags cannot be greater than 10", "BadRequest")
	ErrInvalidTagKey   = Errorf("The TagKey you have provided is invalid", "InvalidTag")
	ErrInvalidTagValue = Errorf("The TagValue you have provided is invalid", "InvalidTag")
	ErrInvalidTag      = Errorf("Cannot provide multiple Tags with the same key", "InvalidTag")
)

type Tagging struct {
	XMLName xml.Name `xml:"Tagging"`
	TagSet  TagSet   `xml:"TagSet"`
}

func (t Tagging) Validate() error {
	if len(t.TagSet.Tags) > maxTags {
		return ErrTooManyTags
	}
	for _, ts := range t.TagSet.Tags {
		if err := ts.Validate(); err != nil {
			return err
		}
	}
	if t.TagSet.ContainsDuplicateTag() {
		return ErrInvalidTag
	}
	return nil
}

func (t Tagging) String() string {
	var buf bytes.Buffer
	for _, tag := range t.TagSet.Tags {
		if buf.Len() > 0 {
			buf.WriteString("&")
		}
		buf.WriteString(tag.String())
	}
	return buf.String()
}

func FromString(tagStr string) (Tagging, error) {
	tags, err := url.ParseQuery(tagStr)
	if err != nil {
		return Tagging{}, err
	}
	var idx = 0
	parsedTags := make([]Tag, len(tags))
	for k := range tags {
		parsedTags[idx].Key = k
		parsedTags[idx].Value = tags.Get(k)
		idx++
	}
	return Tagging{
		TagSet: TagSet{
			Tags: parsedTags,
		},
	}, nil
}

func ParseTagging(reader io.Reader) (*Tagging, error) {
	var t Tagging
	if err := xml.NewDecoder(reader).Decode(&t); err != nil {
		return nil, err
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return &t, nil
}
