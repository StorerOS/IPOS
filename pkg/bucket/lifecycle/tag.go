package lifecycle

import (
	"encoding/xml"
)

type Tag struct {
	XMLName xml.Name `xml:"Tag"`
	Key     string   `xml:"Key,omitempty"`
	Value   string   `xml:"Value,omitempty"`
}

var errTagUnsupported = Errorf("Specifying <Tag></Tag> is not supported")

func (t Tag) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return errTagUnsupported
}

func (t Tag) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return nil
}
