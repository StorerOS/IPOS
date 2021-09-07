package lifecycle

import (
	"encoding/xml"
)

type NoncurrentVersionExpiration struct {
	XMLName        xml.Name `xml:"NoncurrentVersionExpiration"`
	NoncurrentDays int      `xml:"NoncurrentDays,omitempty"`
}

type NoncurrentVersionTransition struct {
	NoncurrentDays int    `xml:"NoncurrentDays"`
	StorageClass   string `xml:"StorageClass"`
}

var (
	errNoncurrentVersionExpirationUnsupported = Errorf("Specifying <NoncurrentVersionExpiration></NoncurrentVersionExpiration> is not supported")
	errNoncurrentVersionTransitionUnsupported = Errorf("Specifying <NoncurrentVersionTransition></NoncurrentVersionTransition> is not supported")
)

func (n NoncurrentVersionExpiration) UnmarshalXML(d *xml.Decoder, startElement xml.StartElement) error {
	return errNoncurrentVersionExpirationUnsupported
}

func (n NoncurrentVersionTransition) UnmarshalXML(d *xml.Decoder, startElement xml.StartElement) error {
	return errNoncurrentVersionTransitionUnsupported
}

func (n NoncurrentVersionTransition) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return nil
}

func (n NoncurrentVersionExpiration) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return nil
}
