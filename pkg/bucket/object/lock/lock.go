package lock

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/env"
	"github.com/beevik/ntp"
)

type RetMode string

const (
	RetGovernance RetMode = "GOVERNANCE"

	RetCompliance RetMode = "COMPLIANCE"
)

func (r RetMode) Valid() bool {
	switch r {
	case RetGovernance, RetCompliance:
		return true
	}
	return false
}

func parseRetMode(modeStr string) (mode RetMode) {
	switch strings.ToUpper(modeStr) {
	case "GOVERNANCE":
		mode = RetGovernance
	case "COMPLIANCE":
		mode = RetCompliance
	}
	return mode
}

type LegalHoldStatus string

const (
	LegalHoldOn LegalHoldStatus = "ON"

	LegalHoldOff LegalHoldStatus = "OFF"
)

func (l LegalHoldStatus) Valid() bool {
	switch l {
	case LegalHoldOn, LegalHoldOff:
		return true
	}
	return false
}

func parseLegalHoldStatus(holdStr string) (st LegalHoldStatus) {
	switch strings.ToUpper(holdStr) {
	case "ON":
		st = LegalHoldOn
	case "OFF":
		st = LegalHoldOff
	}
	return st
}

const (
	AmzObjectLockBypassRetGovernance = "X-Amz-Bypass-Governance-Retention"
	AmzObjectLockRetainUntilDate     = "X-Amz-Object-Lock-Retain-Until-Date"
	AmzObjectLockMode                = "X-Amz-Object-Lock-Mode"
	AmzObjectLockLegalHold           = "X-Amz-Object-Lock-Legal-Hold"
)

var (
	ErrMalformedBucketObjectConfig = errors.New("invalid bucket object lock config")
	ErrInvalidRetentionDate        = errors.New("date must be provided in ISO 8601 format")
	ErrPastObjectLockRetainDate    = errors.New("the retain until date must be in the future")
	ErrUnknownWORMModeDirective    = errors.New("unknown WORM mode directive")
	ErrObjectLockMissingContentMD5 = errors.New("content-MD5 HTTP header is required for Put Object requests with Object Lock parameters")
	ErrObjectLockInvalidHeaders    = errors.New("x-amz-object-lock-retain-until-date and x-amz-object-lock-mode must both be supplied")
	ErrMalformedXML                = errors.New("the XML you provided was not well-formed or did not validate against our published schema")
)

const (
	ntpServerEnv = "IPOS_NTP_SERVER"
)

var (
	ntpServer = env.Get(ntpServerEnv, "")
)

func UTCNowNTP() (time.Time, error) {
	if ntpServer == "" {
		return time.Now().UTC(), nil
	}
	return ntp.Time(ntpServer)
}

type Retention struct {
	Mode     RetMode
	Validity time.Duration
}

func (r Retention) IsEmpty() bool {
	return !r.Mode.Valid() || r.Validity == 0
}

func (r Retention) Retain(created time.Time) bool {
	t, err := UTCNowNTP()
	if err != nil {
		logger.LogIf(context.Background(), err)
		return true
	}
	return created.Add(r.Validity).After(t)
}

type BucketObjectLockConfig struct {
	sync.RWMutex
	retentionMap map[string]Retention
}

func (config *BucketObjectLockConfig) Set(bucketName string, retention Retention) {
	config.Lock()
	config.retentionMap[bucketName] = retention
	config.Unlock()
}

func (config *BucketObjectLockConfig) Get(bucketName string) (r Retention, ok bool) {
	config.RLock()
	defer config.RUnlock()
	r, ok = config.retentionMap[bucketName]
	return r, ok
}

func (config *BucketObjectLockConfig) Remove(bucketName string) {
	config.Lock()
	delete(config.retentionMap, bucketName)
	config.Unlock()
}

func NewBucketObjectLockConfig() *BucketObjectLockConfig {
	return &BucketObjectLockConfig{
		retentionMap: map[string]Retention{},
	}
}

type DefaultRetention struct {
	XMLName xml.Name `xml:"DefaultRetention"`
	Mode    RetMode  `xml:"Mode"`
	Days    *uint64  `xml:"Days"`
	Years   *uint64  `xml:"Years"`
}

const (
	maximumRetentionDays  = 36500
	maximumRetentionYears = 100
)

func (dr *DefaultRetention) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type defaultRetention DefaultRetention
	retention := defaultRetention{}

	if err := d.DecodeElement(&retention, &start); err != nil {
		return err
	}

	switch retention.Mode {
	case RetGovernance, RetCompliance:
	default:
		return fmt.Errorf("unknown retention mode %v", retention.Mode)
	}

	if retention.Days == nil && retention.Years == nil {
		return fmt.Errorf("either Days or Years must be specified")
	}

	if retention.Days != nil && retention.Years != nil {
		return fmt.Errorf("either Days or Years must be specified, not both")
	}

	if retention.Days != nil {
		if *retention.Days == 0 {
			return fmt.Errorf("Default retention period must be a positive integer value for 'Days'")
		}
		if *retention.Days > maximumRetentionDays {
			return fmt.Errorf("Default retention period too large for 'Days' %d", *retention.Days)
		}
	} else if *retention.Years == 0 {
		return fmt.Errorf("Default retention period must be a positive integer value for 'Years'")
	} else if *retention.Years > maximumRetentionYears {
		return fmt.Errorf("Default retention period too large for 'Years' %d", *retention.Years)
	}

	*dr = DefaultRetention(retention)

	return nil
}

type Config struct {
	XMLNS             string   `xml:"xmlns,attr,omitempty"`
	XMLName           xml.Name `xml:"ObjectLockConfiguration"`
	ObjectLockEnabled string   `xml:"ObjectLockEnabled"`
	Rule              *struct {
		DefaultRetention DefaultRetention `xml:"DefaultRetention"`
	} `xml:"Rule,omitempty"`
}

func (config *Config) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type objectLockConfig Config
	parsedConfig := objectLockConfig{}

	if err := d.DecodeElement(&parsedConfig, &start); err != nil {
		return err
	}

	if parsedConfig.ObjectLockEnabled != "Enabled" {
		return fmt.Errorf("only 'Enabled' value is allowd to ObjectLockEnabled element")
	}

	*config = Config(parsedConfig)
	return nil
}

func (config *Config) ToRetention() (r Retention) {
	if config.Rule != nil {
		r.Mode = config.Rule.DefaultRetention.Mode

		t, err := UTCNowNTP()
		if err != nil {
			logger.LogIf(context.Background(), err)
			return r
		}

		if config.Rule.DefaultRetention.Days != nil {
			r.Validity = t.AddDate(0, 0, int(*config.Rule.DefaultRetention.Days)).Sub(t)
		} else {
			r.Validity = t.AddDate(int(*config.Rule.DefaultRetention.Years), 0, 0).Sub(t)
		}
	}

	return r
}

const maxObjectLockConfigSize = 1 << 12

func ParseObjectLockConfig(reader io.Reader) (*Config, error) {
	config := Config{}
	if err := xml.NewDecoder(io.LimitReader(reader, maxObjectLockConfigSize)).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func NewObjectLockConfig() *Config {
	return &Config{
		ObjectLockEnabled: "Enabled",
	}
}

type RetentionDate struct {
	time.Time
}

func (rDate *RetentionDate) UnmarshalXML(d *xml.Decoder, startElement xml.StartElement) error {
	var dateStr string
	err := d.DecodeElement(&dateStr, &startElement)
	if err != nil {
		return err
	}
	retDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return ErrInvalidRetentionDate
	}

	*rDate = RetentionDate{retDate}
	return nil
}

func (rDate *RetentionDate) MarshalXML(e *xml.Encoder, startElement xml.StartElement) error {
	if *rDate == (RetentionDate{time.Time{}}) {
		return nil
	}
	return e.EncodeElement(rDate.Format(time.RFC3339), startElement)
}

type ObjectRetention struct {
	XMLNS           string        `xml:"xmlns,attr,omitempty"`
	XMLName         xml.Name      `xml:"Retention"`
	Mode            RetMode       `xml:"Mode,omitempty"`
	RetainUntilDate RetentionDate `xml:"RetainUntilDate,omitempty"`
}

const maxObjectRetentionSize = 1 << 12

func ParseObjectRetention(reader io.Reader) (*ObjectRetention, error) {
	ret := ObjectRetention{}
	if err := xml.NewDecoder(io.LimitReader(reader, maxObjectRetentionSize)).Decode(&ret); err != nil {
		return nil, err
	}
	if !ret.Mode.Valid() {
		return &ret, ErrUnknownWORMModeDirective
	}

	t, err := UTCNowNTP()
	if err != nil {
		logger.LogIf(context.Background(), err)
		return &ret, ErrPastObjectLockRetainDate
	}

	if ret.RetainUntilDate.Before(t) {
		return &ret, ErrPastObjectLockRetainDate
	}

	return &ret, nil
}

func IsObjectLockRetentionRequested(h http.Header) bool {
	if _, ok := h[AmzObjectLockMode]; ok {
		return true
	}
	if _, ok := h[AmzObjectLockRetainUntilDate]; ok {
		return true
	}
	return false
}

func IsObjectLockLegalHoldRequested(h http.Header) bool {
	_, ok := h[AmzObjectLockLegalHold]
	return ok
}

func IsObjectLockGovernanceBypassSet(h http.Header) bool {
	return strings.ToLower(h.Get(AmzObjectLockBypassRetGovernance)) == "true"
}

func IsObjectLockRequested(h http.Header) bool {
	return IsObjectLockLegalHoldRequested(h) || IsObjectLockRetentionRequested(h)
}

func ParseObjectLockRetentionHeaders(h http.Header) (rmode RetMode, r RetentionDate, err error) {
	retMode := h.Get(AmzObjectLockMode)
	dateStr := h.Get(AmzObjectLockRetainUntilDate)
	if len(retMode) == 0 || len(dateStr) == 0 {
		return rmode, r, ErrObjectLockInvalidHeaders
	}

	rmode = parseRetMode(retMode)
	if !rmode.Valid() {
		return rmode, r, ErrUnknownWORMModeDirective
	}

	var retDate time.Time
	retDate, err = time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return rmode, r, ErrInvalidRetentionDate
	}

	t, err := UTCNowNTP()
	if err != nil {
		logger.LogIf(context.Background(), err)
		return rmode, r, ErrPastObjectLockRetainDate
	}

	if retDate.Before(t) {
		return rmode, r, ErrPastObjectLockRetainDate
	}

	return rmode, RetentionDate{retDate}, nil

}

func GetObjectRetentionMeta(meta map[string]string) ObjectRetention {
	var mode RetMode
	var retainTill RetentionDate

	var modeStr, tillStr string
	ok := false

	modeStr, ok = meta[strings.ToLower(AmzObjectLockMode)]
	if !ok {
		modeStr, ok = meta[AmzObjectLockMode]
	}
	if ok {
		mode = parseRetMode(modeStr)
	}
	tillStr, ok = meta[strings.ToLower(AmzObjectLockRetainUntilDate)]
	if !ok {
		tillStr, ok = meta[AmzObjectLockRetainUntilDate]
	}
	if ok {
		if t, e := time.Parse(time.RFC3339, tillStr); e == nil {
			retainTill = RetentionDate{t.UTC()}
		}
	}
	return ObjectRetention{XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Mode: mode, RetainUntilDate: retainTill}
}

func GetObjectLegalHoldMeta(meta map[string]string) ObjectLegalHold {
	holdStr, ok := meta[strings.ToLower(AmzObjectLockLegalHold)]
	if !ok {
		holdStr, ok = meta[AmzObjectLockLegalHold]
	}
	if ok {
		return ObjectLegalHold{XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Status: parseLegalHoldStatus(holdStr)}
	}
	return ObjectLegalHold{}
}

func ParseObjectLockLegalHoldHeaders(h http.Header) (lhold ObjectLegalHold, err error) {
	holdStatus, ok := h[AmzObjectLockLegalHold]
	if ok {
		lh := parseLegalHoldStatus(holdStatus[0])
		if !lh.Valid() {
			return lhold, ErrUnknownWORMModeDirective
		}
		lhold = ObjectLegalHold{XMLNS: "http://s3.amazonaws.com/doc/2006-03-01/", Status: lh}
	}
	return lhold, nil

}

type ObjectLegalHold struct {
	XMLNS   string          `xml:"xmlns,attr,omitempty"`
	XMLName xml.Name        `xml:"LegalHold"`
	Status  LegalHoldStatus `xml:"Status,omitempty"`
}

func (l *ObjectLegalHold) IsEmpty() bool {
	return !l.Status.Valid()
}

func ParseObjectLegalHold(reader io.Reader) (hold *ObjectLegalHold, err error) {
	hold = &ObjectLegalHold{}
	if err = xml.NewDecoder(reader).Decode(hold); err != nil {
		return
	}

	if !hold.Status.Valid() {
		return nil, ErrMalformedXML
	}
	return
}

func FilterObjectLockMetadata(metadata map[string]string, filterRetention, filterLegalHold bool) map[string]string {
	dst := metadata
	var copied bool
	delKey := func(key string) {
		if _, ok := metadata[key]; !ok {
			return
		}
		if !copied {
			dst = make(map[string]string, len(metadata))
			for k, v := range metadata {
				dst[k] = v
			}
			copied = true
		}
		delete(dst, key)
	}
	legalHold := GetObjectLegalHoldMeta(metadata)
	if !legalHold.Status.Valid() || filterLegalHold {
		delKey(AmzObjectLockLegalHold)
	}

	ret := GetObjectRetentionMeta(metadata)
	if !ret.Mode.Valid() || filterRetention {
		delKey(AmzObjectLockMode)
		delKey(AmzObjectLockRetainUntilDate)
		return dst
	}
	return dst
}
