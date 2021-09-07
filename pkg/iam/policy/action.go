package iampolicy

import (
	"encoding/json"

	"github.com/storeros/ipos/pkg/bucket/policy/condition"
	"github.com/storeros/ipos/pkg/wildcard"
)

type Action string

const (
	AbortMultipartUploadAction Action = "s3:AbortMultipartUpload"

	CreateBucketAction = "s3:CreateBucket"

	DeleteBucketAction = "s3:DeleteBucket"

	ForceDeleteBucketAction = "s3:ForceDeleteBucket"

	DeleteBucketPolicyAction = "s3:DeleteBucketPolicy"

	DeleteObjectAction = "s3:DeleteObject"

	GetBucketLocationAction = "s3:GetBucketLocation"

	GetBucketNotificationAction = "s3:GetBucketNotification"

	GetBucketPolicyAction = "s3:GetBucketPolicy"

	GetObjectAction = "s3:GetObject"

	HeadBucketAction = "s3:HeadBucket"

	ListAllMyBucketsAction = "s3:ListAllMyBuckets"

	ListBucketAction = "s3:ListBucket"

	ListBucketMultipartUploadsAction = "s3:ListBucketMultipartUploads"

	ListenBucketNotificationAction = "s3:ListenBucketNotification"

	ListMultipartUploadPartsAction = "s3:ListMultipartUploadParts"

	PutBucketLifecycleAction = "s3:PutBucketLifecycle"

	GetBucketLifecycleAction = "s3:GetBucketLifecycle"

	PutBucketNotificationAction = "s3:PutBucketNotification"

	PutBucketPolicyAction = "s3:PutBucketPolicy"

	PutObjectAction = "s3:PutObject"

	BypassGovernanceRetentionAction = "s3:BypassGovernanceRetention"

	PutObjectRetentionAction = "s3:PutObjectRetention"

	GetObjectRetentionAction = "s3:GetObjectRetention"

	GetObjectLegalHoldAction = "s3:GetObjectLegalHold"

	PutObjectLegalHoldAction = "s3:PutObjectLegalHold"

	GetBucketObjectLockConfigurationAction = "s3:GetBucketObjectLockConfiguration"

	PutBucketObjectLockConfigurationAction = "s3:PutBucketObjectLockConfiguration"

	GetObjectTaggingAction = "s3:GetObjectTagging"

	PutObjectTaggingAction = "s3:PutObjectTagging"

	DeleteObjectTaggingAction = "s3:DeleteObjectTagging"

	PutBucketEncryptionAction = "s3:PutEncryptionConfiguration"

	GetBucketEncryptionAction = "s3:GetEncryptionConfiguration"

	AllActions = "s3:*"
)

var supportedActions = map[Action]struct{}{
	AllActions:                             {},
	AbortMultipartUploadAction:             {},
	CreateBucketAction:                     {},
	DeleteBucketAction:                     {},
	ForceDeleteBucketAction:                {},
	DeleteBucketPolicyAction:               {},
	DeleteObjectAction:                     {},
	GetBucketLocationAction:                {},
	GetBucketNotificationAction:            {},
	GetBucketPolicyAction:                  {},
	GetObjectAction:                        {},
	HeadBucketAction:                       {},
	ListAllMyBucketsAction:                 {},
	ListBucketAction:                       {},
	ListBucketMultipartUploadsAction:       {},
	ListenBucketNotificationAction:         {},
	ListMultipartUploadPartsAction:         {},
	PutBucketNotificationAction:            {},
	PutBucketPolicyAction:                  {},
	PutObjectAction:                        {},
	GetBucketLifecycleAction:               {},
	PutBucketLifecycleAction:               {},
	PutObjectRetentionAction:               {},
	GetObjectRetentionAction:               {},
	GetObjectLegalHoldAction:               {},
	PutObjectLegalHoldAction:               {},
	PutBucketObjectLockConfigurationAction: {},
	GetBucketObjectLockConfigurationAction: {},
	BypassGovernanceRetentionAction:        {},
	GetObjectTaggingAction:                 {},
	PutObjectTaggingAction:                 {},
	DeleteObjectTaggingAction:              {},
	PutBucketEncryptionAction:              {},
	GetBucketEncryptionAction:              {},
}

var supportedObjectActions = map[Action]struct{}{
	AllActions:                      {},
	AbortMultipartUploadAction:      {},
	DeleteObjectAction:              {},
	GetObjectAction:                 {},
	ListMultipartUploadPartsAction:  {},
	PutObjectAction:                 {},
	BypassGovernanceRetentionAction: {},
	PutObjectRetentionAction:        {},
	GetObjectRetentionAction:        {},
	PutObjectLegalHoldAction:        {},
	GetObjectLegalHoldAction:        {},
	GetObjectTaggingAction:          {},
	PutObjectTaggingAction:          {},
	DeleteObjectTaggingAction:       {},
}

func (action Action) isObjectAction() bool {
	_, ok := supportedObjectActions[action]
	return ok
}

func (action Action) Match(a Action) bool {
	return wildcard.Match(string(action), string(a))
}

func (action Action) IsValid() bool {
	_, ok := supportedActions[action]
	return ok
}

func (action Action) MarshalJSON() ([]byte, error) {
	if action.IsValid() || AdminAction(action).IsValid() {
		return json.Marshal(string(action))
	}

	return nil, Errorf("invalid action '%v'", action)
}

func (action *Action) UnmarshalJSON(data []byte) error {
	var s string

	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	a := Action(s)
	if !a.IsValid() {
		return Errorf("invalid action '%v'", s)
	}

	*action = a

	return nil
}

func parseAction(s string) (Action, error) {
	adminAction, err := parseAdminAction(s)
	if err == nil {
		return Action(adminAction), nil
	}
	action := Action(s)

	if action.IsValid() {
		return action, nil
	}

	return action, Errorf("unsupported action '%v'", s)
}

var actionConditionKeyMap = map[Action]condition.KeySet{
	AllActions: condition.NewKeySet(condition.AllSupportedKeys...),

	AbortMultipartUploadAction: condition.NewKeySet(condition.CommonKeys...),

	CreateBucketAction: condition.NewKeySet(condition.CommonKeys...),

	DeleteBucketPolicyAction: condition.NewKeySet(condition.CommonKeys...),

	DeleteObjectAction: condition.NewKeySet(condition.CommonKeys...),

	GetBucketLocationAction: condition.NewKeySet(condition.CommonKeys...),

	GetBucketNotificationAction: condition.NewKeySet(condition.CommonKeys...),

	GetBucketPolicyAction: condition.NewKeySet(condition.CommonKeys...),

	GetObjectAction: condition.NewKeySet(
		append([]condition.Key{
			condition.S3XAmzServerSideEncryption,
			condition.S3XAmzServerSideEncryptionCustomerAlgorithm,
			condition.S3XAmzStorageClass,
		}, condition.CommonKeys...)...),

	HeadBucketAction: condition.NewKeySet(condition.CommonKeys...),

	ListAllMyBucketsAction: condition.NewKeySet(condition.CommonKeys...),

	ListBucketAction: condition.NewKeySet(
		append([]condition.Key{
			condition.S3Prefix,
			condition.S3Delimiter,
			condition.S3MaxKeys,
		}, condition.CommonKeys...)...),

	ListBucketMultipartUploadsAction: condition.NewKeySet(condition.CommonKeys...),

	ListenBucketNotificationAction: condition.NewKeySet(condition.CommonKeys...),

	ListMultipartUploadPartsAction: condition.NewKeySet(condition.CommonKeys...),

	PutBucketNotificationAction: condition.NewKeySet(condition.CommonKeys...),

	PutBucketPolicyAction: condition.NewKeySet(condition.CommonKeys...),

	PutObjectAction: condition.NewKeySet(
		append([]condition.Key{
			condition.S3XAmzCopySource,
			condition.S3XAmzServerSideEncryption,
			condition.S3XAmzServerSideEncryptionCustomerAlgorithm,
			condition.S3XAmzMetadataDirective,
			condition.S3XAmzStorageClass,
			condition.S3ObjectLockRetainUntilDate,
			condition.S3ObjectLockMode,
			condition.S3ObjectLockLegalHold,
		}, condition.CommonKeys...)...),

	PutObjectRetentionAction: condition.NewKeySet(
		append([]condition.Key{
			condition.S3ObjectLockRemainingRetentionDays,
			condition.S3ObjectLockRetainUntilDate,
			condition.S3ObjectLockMode,
		}, condition.CommonKeys...)...),

	GetObjectRetentionAction: condition.NewKeySet(condition.CommonKeys...),
	PutObjectLegalHoldAction: condition.NewKeySet(
		append([]condition.Key{
			condition.S3ObjectLockLegalHold,
		}, condition.CommonKeys...)...),
	GetObjectLegalHoldAction: condition.NewKeySet(condition.CommonKeys...),

	BypassGovernanceRetentionAction: condition.NewKeySet(
		append([]condition.Key{
			condition.S3ObjectLockRemainingRetentionDays,
			condition.S3ObjectLockRetainUntilDate,
			condition.S3ObjectLockMode,
			condition.S3ObjectLockLegalHold,
		}, condition.CommonKeys...)...),

	GetBucketObjectLockConfigurationAction: condition.NewKeySet(condition.CommonKeys...),
	PutBucketObjectLockConfigurationAction: condition.NewKeySet(condition.CommonKeys...),
	PutObjectTaggingAction:                 condition.NewKeySet(condition.CommonKeys...),
	GetObjectTaggingAction:                 condition.NewKeySet(condition.CommonKeys...),
	DeleteObjectTaggingAction:              condition.NewKeySet(condition.CommonKeys...),
}
