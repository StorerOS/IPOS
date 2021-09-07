package tagging

import (
	"encoding/xml"
	"strings"
	"unicode/utf8"
)

type Tag struct {
	XMLName xml.Name `xml:"Tag"`
	Key     string   `xml:"Key,omitempty"`
	Value   string   `xml:"Value,omitempty"`
}

func (t Tag) Validate() error {
	if err := t.validateKey(); err != nil {
		return err
	}
	if err := t.validateValue(); err != nil {
		return err
	}
	return nil
}

func (t Tag) validateKey() error {
	if utf8.RuneCountInString(t.Key) > maxTagKeyLength {
		return ErrInvalidTagKey
	}
	if len(t.Key) == 0 {
		return ErrInvalidTagKey
	}
	if strings.Contains(t.Key, "&") {
		return ErrInvalidTagKey
	}
	return nil
}

func (t Tag) validateValue() error {
	if utf8.RuneCountInString(t.Value) > maxTagValueLength {
		return ErrInvalidTagValue
	}
	if strings.Contains(t.Value, "&") {
		return ErrInvalidTagValue
	}
	return nil
}

func (t Tag) IsEmpty() bool {
	return t.Key == "" && t.Value == ""
}

func (t Tag) String() string {
	return t.Key + "=" + t.Value
}
