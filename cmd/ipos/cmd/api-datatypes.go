package cmd

import (
	"encoding/xml"
)

type ObjectIdentifier struct {
	ObjectName string `xml:"Key"`
}

type createBucketLocationConfiguration struct {
	XMLName  xml.Name `xml:"CreateBucketConfiguration" json:"-"`
	Location string   `xml:"LocationConstraint"`
}

type DeleteObjectsRequest struct {
	Quiet   bool
	Objects []ObjectIdentifier `xml:"Object"`
}
