package lifecycle

import (
	"encoding/xml"
	"time"
)

var (
	errLifecycleInvalidDate       = Errorf("Date must be provided in ISO 8601 format")
	errLifecycleInvalidDays       = Errorf("Days must be positive integer when used with Expiration")
	errLifecycleInvalidExpiration = Errorf("At least one of Days or Date should be present inside Expiration")
	errLifecycleDateNotMidnight   = Errorf("'Date' must be at midnight GMT")
)

type ExpirationDays int

func (eDays *ExpirationDays) UnmarshalXML(d *xml.Decoder, startElement xml.StartElement) error {
	var numDays int
	err := d.DecodeElement(&numDays, &startElement)
	if err != nil {
		return err
	}
	if numDays <= 0 {
		return errLifecycleInvalidDays
	}
	*eDays = ExpirationDays(numDays)
	return nil
}

func (eDays *ExpirationDays) MarshalXML(e *xml.Encoder, startElement xml.StartElement) error {
	if *eDays == ExpirationDays(0) {
		return nil
	}
	return e.EncodeElement(int(*eDays), startElement)
}

type ExpirationDate struct {
	time.Time
}

func (eDate *ExpirationDate) UnmarshalXML(d *xml.Decoder, startElement xml.StartElement) error {
	var dateStr string
	err := d.DecodeElement(&dateStr, &startElement)
	if err != nil {
		return err
	}
	expDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return errLifecycleInvalidDate
	}
	hr, min, sec := expDate.Clock()
	nsec := expDate.Nanosecond()
	loc := expDate.Location()
	if !(hr == 0 && min == 0 && sec == 0 && nsec == 0 && loc.String() == time.UTC.String()) {
		return errLifecycleDateNotMidnight
	}

	*eDate = ExpirationDate{expDate}
	return nil
}

func (eDate *ExpirationDate) MarshalXML(e *xml.Encoder, startElement xml.StartElement) error {
	if *eDate == (ExpirationDate{time.Time{}}) {
		return nil
	}
	return e.EncodeElement(eDate.Format(time.RFC3339), startElement)
}

type Expiration struct {
	XMLName xml.Name       `xml:"Expiration"`
	Days    ExpirationDays `xml:"Days,omitempty"`
	Date    ExpirationDate `xml:"Date,omitempty"`
}

func (e Expiration) Validate() error {
	if e.IsDaysNull() && e.IsDateNull() {
		return errLifecycleInvalidExpiration
	}

	if !e.IsDaysNull() && !e.IsDateNull() {
		return errLifecycleInvalidExpiration
	}
	return nil
}

func (e Expiration) IsDaysNull() bool {
	return e.Days == ExpirationDays(0)
}

func (e Expiration) IsDateNull() bool {
	return e.Date == ExpirationDate{time.Time{}}
}

func (e Expiration) IsNull() bool {
	return e.IsDaysNull() && e.IsDateNull()
}
