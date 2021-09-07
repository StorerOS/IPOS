package cmd

import (
	"encoding/xml"
	"errors"
	"io"
)

const (
	AES256 SSEAlgorithm = "AES256"
	AWSKms SSEAlgorithm = "aws:kms"
)

type SSEAlgorithm string

func (alg *SSEAlgorithm) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}

	switch s {
	case string(AES256):
		*alg = AES256
	case string(AWSKms):
		*alg = AWSKms
	default:
		return errors.New("Unknown SSE algorithm")
	}

	return nil
}

func (alg *SSEAlgorithm) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(string(*alg), start)
}

type EncryptionAction struct {
	Algorithm   SSEAlgorithm `xml:"SSEAlgorithm,omitempty"`
	MasterKeyID string       `xml:"KMSMasterKeyID,omitempty"`
}

type SSERule struct {
	DefaultEncryptionAction EncryptionAction `xml:"ApplyServerSideEncryptionByDefault"`
}

const xmlNS = "http://s3.amazonaws.com/doc/2006-03-01/"

type BucketSSEConfig struct {
	XMLNS   string    `xml:"xmlns,attr,omitempty"`
	XMLName xml.Name  `xml:"ServerSideEncryptionConfiguration"`
	Rules   []SSERule `xml:"Rule"`
}

func ParseBucketSSEConfig(r io.Reader) (*BucketSSEConfig, error) {
	var config BucketSSEConfig
	err := xml.NewDecoder(r).Decode(&config)
	if err != nil {
		return nil, err
	}

	if len(config.Rules) != 1 {
		return nil, errors.New("Only one server-side encryption rule is allowed")
	}

	for _, rule := range config.Rules {
		switch rule.DefaultEncryptionAction.Algorithm {
		case AES256:
			if rule.DefaultEncryptionAction.MasterKeyID != "" {
				return nil, errors.New("MasterKeyID is allowed with aws:kms only")
			}
		case AWSKms:
			if rule.DefaultEncryptionAction.MasterKeyID == "" {
				return nil, errors.New("MasterKeyID is missing")
			}
		}
	}

	if config.XMLNS == "" {
		config.XMLNS = xmlNS
	}

	return &config, nil
}
