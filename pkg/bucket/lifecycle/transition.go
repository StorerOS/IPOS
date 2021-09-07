package lifecycle

import (
	"encoding/xml"
)

type Transition struct {
	XMLName      xml.Name `xml:"Transition"`
	Days         int      `xml:"Days,omitempty"`
	Date         string   `xml:"Date,omitempty"`
	StorageClass string   `xml:"StorageClass"`
}

var errTransitionUnsupported = Errorf("Specifying <Transition></Transition> tag is not supported")

func (t Transition) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return errTransitionUnsupported
}

func (t Transition) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return nil
}
