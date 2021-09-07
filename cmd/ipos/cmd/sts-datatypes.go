package cmd

import (
	"encoding/xml"

	"github.com/storeros/ipos/pkg/auth"
)

type AssumedRoleUser struct {
	Arn string

	AssumedRoleID string `xml:"AssumeRoleId"`
}

type AssumeRoleResponse struct {
	XMLName xml.Name `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleResponse" json:"-"`

	Result           AssumeRoleResult `xml:"AssumeRoleResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

type AssumeRoleResult struct {
	AssumedRoleUser AssumedRoleUser `xml:",omitempty"`

	Credentials auth.Credentials `xml:",omitempty"`

	PackedPolicySize int `xml:",omitempty"`
}

type AssumeRoleWithWebIdentityResponse struct {
	XMLName          xml.Name          `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleWithWebIdentityResponse" json:"-"`
	Result           WebIdentityResult `xml:"AssumeRoleWithWebIdentityResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

type WebIdentityResult struct {
	AssumedRoleUser AssumedRoleUser `xml:",omitempty"`

	Audience string `xml:",omitempty"`

	Credentials auth.Credentials `xml:",omitempty"`

	PackedPolicySize int `xml:",omitempty"`

	Provider string `xml:",omitempty"`

	SubjectFromWebIdentityToken string `xml:",omitempty"`
}

type AssumeRoleWithClientGrantsResponse struct {
	XMLName          xml.Name           `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleWithClientGrantsResponse" json:"-"`
	Result           ClientGrantsResult `xml:"AssumeRoleWithClientGrantsResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

type ClientGrantsResult struct {
	AssumedRoleUser AssumedRoleUser `xml:",omitempty"`

	Audience string `xml:",omitempty"`

	Credentials auth.Credentials `xml:",omitempty"`

	PackedPolicySize int `xml:",omitempty"`

	Provider string `xml:",omitempty"`

	SubjectFromToken string `xml:",omitempty"`
}

type AssumeRoleWithLDAPResponse struct {
	XMLName          xml.Name           `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleWithLDAPIdentityResponse" json:"-"`
	Result           LDAPIdentityResult `xml:"AssumeRoleWithLDAPIdentityResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

type LDAPIdentityResult struct {
	Credentials auth.Credentials `xml:",omitempty"`
}
