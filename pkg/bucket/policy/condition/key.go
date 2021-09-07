package condition

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Key string

const (
	S3XAmzCopySource Key = "s3:x-amz-copy-source"

	S3XAmzServerSideEncryption Key = "s3:x-amz-server-side-encryption"

	S3XAmzServerSideEncryptionCustomerAlgorithm Key = "s3:x-amz-server-side-encryption-customer-algorithm"

	S3XAmzMetadataDirective Key = "s3:x-amz-metadata-directive"

	S3XAmzContentSha256 = "s3:x-amz-content-sha256"

	S3XAmzStorageClass Key = "s3:x-amz-storage-class"

	S3LocationConstraint Key = "s3:LocationConstraint"

	S3Prefix Key = "s3:prefix"

	S3Delimiter Key = "s3:delimiter"

	S3MaxKeys Key = "s3:max-keys"

	S3ObjectLockRemainingRetentionDays Key = "s3:object-lock-remaining-retention-days"

	S3ObjectLockMode Key = "s3:object-lock-mode"

	S3ObjectLockRetainUntilDate Key = "s3:object-lock-retain-until-date"

	S3ObjectLockLegalHold Key = "s3:object-lock-legal-hold"

	AWSReferer Key = "aws:Referer"

	AWSSourceIP Key = "aws:SourceIp"

	AWSUserAgent Key = "aws:UserAgent"

	AWSSecureTransport Key = "aws:SecureTransport"

	AWSCurrentTime Key = "aws:CurrentTime"

	AWSEpochTime Key = "aws:EpochTime"

	AWSPrincipalType Key = "aws:principaltype"

	AWSUserID Key = "aws:userid"

	AWSUsername Key = "aws:username"
)

var AllSupportedKeys = append([]Key{
	S3XAmzCopySource,
	S3XAmzServerSideEncryption,
	S3XAmzServerSideEncryptionCustomerAlgorithm,
	S3XAmzMetadataDirective,
	S3XAmzStorageClass,
	S3XAmzContentSha256,
	S3LocationConstraint,
	S3Prefix,
	S3Delimiter,
	S3MaxKeys,
	S3ObjectLockRemainingRetentionDays,
	S3ObjectLockMode,
	S3ObjectLockLegalHold,
	S3ObjectLockRetainUntilDate,
	AWSReferer,
	AWSSourceIP,
	AWSUserAgent,
	AWSSecureTransport,
	AWSCurrentTime,
	AWSEpochTime,
	AWSPrincipalType,
	AWSUserID,
	AWSUsername,
}, JWTKeys...)

var CommonKeys = append([]Key{
	AWSReferer,
	AWSSourceIP,
	AWSUserAgent,
	AWSSecureTransport,
	AWSCurrentTime,
	AWSEpochTime,
	AWSPrincipalType,
	AWSUserID,
	AWSUsername,
	S3XAmzContentSha256,
}, JWTKeys...)

func substFuncFromValues(values map[string][]string) func(string) string {
	return func(v string) string {
		for _, key := range CommonKeys {
			if rvalues, ok := values[key.Name()]; ok && rvalues[0] != "" {
				v = strings.Replace(v, key.VarName(), rvalues[0], -1)
			}
		}
		return v
	}
}

func (key Key) IsValid() bool {
	for _, supKey := range AllSupportedKeys {
		if supKey == key {
			return true
		}
	}

	return false
}

func (key Key) MarshalJSON() ([]byte, error) {
	if !key.IsValid() {
		return nil, fmt.Errorf("unknown key %v", key)
	}

	return json.Marshal(string(key))
}

func (key Key) VarName() string {
	return fmt.Sprintf("${%s}", key)
}

func (key Key) Name() string {
	keyString := string(key)

	if strings.HasPrefix(keyString, "aws:") {
		return strings.TrimPrefix(keyString, "aws:")
	} else if strings.HasPrefix(keyString, "jwt:") {
		return strings.TrimPrefix(keyString, "jwt:")
	}
	return strings.TrimPrefix(keyString, "s3:")
}

func (key *Key) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsedKey, err := parseKey(s)
	if err != nil {
		return err
	}

	*key = parsedKey
	return nil
}

func parseKey(s string) (Key, error) {
	key := Key(s)

	if key.IsValid() {
		return key, nil
	}

	return key, fmt.Errorf("invalid condition key '%v'", s)
}

type KeySet map[Key]struct{}

func (set KeySet) Add(key Key) {
	set[key] = struct{}{}
}

func (set KeySet) Difference(sset KeySet) KeySet {
	nset := make(KeySet)

	for k := range set {
		if _, ok := sset[k]; !ok {
			nset.Add(k)
		}
	}

	return nset
}

func (set KeySet) IsEmpty() bool {
	return len(set) == 0
}

func (set KeySet) String() string {
	return fmt.Sprintf("%v", set.ToSlice())
}

func (set KeySet) ToSlice() []Key {
	keys := []Key{}

	for key := range set {
		keys = append(keys, key)
	}

	return keys
}

func NewKeySet(keys ...Key) KeySet {
	set := make(KeySet)
	for _, key := range keys {
		set.Add(key)
	}

	return set
}

var AllSupportedAdminKeys = []Key{
	AWSReferer,
	AWSSourceIP,
	AWSUserAgent,
	AWSSecureTransport,
	AWSCurrentTime,
	AWSEpochTime,
}
