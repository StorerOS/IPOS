package cmd

import (
	"bytes"
	"context"
	"errors"
	"math"
	"net/http"
	"path"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/auth"
	objectlock "github.com/storeros/ipos/pkg/bucket/object/lock"
	"github.com/storeros/ipos/pkg/bucket/policy"
)

func enforceRetentionBypassForDeleteWeb(ctx context.Context, r *http.Request, bucket, object string, getObjectInfoFn GetObjectInfoFn) APIErrorCode {
	opts, err := getOpts(ctx, r, bucket, object)
	if err != nil {
		return toAPIErrorCode(ctx, err)
	}

	oi, err := getObjectInfoFn(ctx, bucket, object, opts)
	if err != nil {
		return toAPIErrorCode(ctx, err)
	}

	lhold := objectlock.GetObjectLegalHoldMeta(oi.UserDefined)
	if lhold.Status.Valid() && lhold.Status == objectlock.LegalHoldOn {
		return ErrObjectLocked
	}

	ret := objectlock.GetObjectRetentionMeta(oi.UserDefined)
	if ret.Mode.Valid() {
		switch ret.Mode {
		case objectlock.RetCompliance:
			t, err := objectlock.UTCNowNTP()
			if err != nil {
				logger.LogIf(ctx, err)
				return ErrObjectLocked
			}

			if !ret.RetainUntilDate.Before(t) {
				return ErrObjectLocked
			}
			return ErrNone
		case objectlock.RetGovernance:
			byPassSet := objectlock.IsObjectLockGovernanceBypassSet(r.Header)
			if !byPassSet {
				t, err := objectlock.UTCNowNTP()
				if err != nil {
					logger.LogIf(ctx, err)
					return ErrObjectLocked
				}

				if !ret.RetainUntilDate.Before(t) {
					return ErrObjectLocked
				}
				return ErrNone
			}
		}
	}
	return ErrNone
}

func enforceRetentionForLifecycle(ctx context.Context, objInfo ObjectInfo) (locked bool) {
	lhold := objectlock.GetObjectLegalHoldMeta(objInfo.UserDefined)
	if lhold.Status.Valid() && lhold.Status == objectlock.LegalHoldOn {
		return true
	}

	ret := objectlock.GetObjectRetentionMeta(objInfo.UserDefined)
	if ret.Mode.Valid() && (ret.Mode == objectlock.RetCompliance || ret.Mode == objectlock.RetGovernance) {
		t, err := objectlock.UTCNowNTP()
		if err != nil {
			logger.LogIf(ctx, err)
			return true
		}
		if ret.RetainUntilDate.After(t) {
			return true
		}
	}
	return false
}

func enforceRetentionBypassForDelete(ctx context.Context, r *http.Request, bucket, object string, getObjectInfoFn GetObjectInfoFn) APIErrorCode {
	opts, err := getOpts(ctx, r, bucket, object)
	if err != nil {
		return toAPIErrorCode(ctx, err)
	}

	oi, err := getObjectInfoFn(ctx, bucket, object, opts)
	if err != nil {
		return toAPIErrorCode(ctx, err)
	}

	lhold := objectlock.GetObjectLegalHoldMeta(oi.UserDefined)
	if lhold.Status.Valid() && lhold.Status == objectlock.LegalHoldOn {
		return ErrObjectLocked
	}

	ret := objectlock.GetObjectRetentionMeta(oi.UserDefined)
	if ret.Mode.Valid() {
		switch ret.Mode {
		case objectlock.RetCompliance:
			t, err := objectlock.UTCNowNTP()
			if err != nil {
				logger.LogIf(ctx, err)
				return ErrObjectLocked
			}

			if !ret.RetainUntilDate.Before(t) {
				return ErrObjectLocked
			}
			return ErrNone
		case objectlock.RetGovernance:
			byPassSet := objectlock.IsObjectLockGovernanceBypassSet(r.Header)
			if !byPassSet {
				t, err := objectlock.UTCNowNTP()
				if err != nil {
					logger.LogIf(ctx, err)
					return ErrObjectLocked
				}

				if !ret.RetainUntilDate.Before(t) {
					return ErrObjectLocked
				}
				return ErrNone
			}
			govBypassPerms1 := checkRequestAuthType(ctx, r, policy.BypassGovernanceRetentionAction, bucket, object)
			govBypassPerms2 := checkRequestAuthType(ctx, r, policy.GetBucketObjectLockConfigurationAction, bucket, object)
			if govBypassPerms1 != ErrNone && govBypassPerms2 != ErrNone {
				return ErrAccessDenied
			}
		}
	}
	return ErrNone
}

func enforceRetentionBypassForPut(ctx context.Context, r *http.Request, bucket, object string, getObjectInfoFn GetObjectInfoFn, objRetention *objectlock.ObjectRetention, cred auth.Credentials, owner bool, claims map[string]interface{}) (ObjectInfo, APIErrorCode) {
	byPassSet := objectlock.IsObjectLockGovernanceBypassSet(r.Header)
	opts, err := getOpts(ctx, r, bucket, object)
	if err != nil {
		return ObjectInfo{}, toAPIErrorCode(ctx, err)
	}

	oi, err := getObjectInfoFn(ctx, bucket, object, opts)
	if err != nil {
		return oi, toAPIErrorCode(ctx, err)
	}

	t, err := objectlock.UTCNowNTP()
	if err != nil {
		logger.LogIf(ctx, err)
		return oi, ErrObjectLocked
	}

	days := int(math.Ceil(math.Abs(objRetention.RetainUntilDate.Sub(t).Hours()) / 24))

	ret := objectlock.GetObjectRetentionMeta(oi.UserDefined)
	if ret.Mode.Valid() {
		if ret.RetainUntilDate.Before(t) {
			perm := isPutRetentionAllowed(bucket, object,
				days, objRetention.RetainUntilDate.Time,
				objRetention.Mode, byPassSet, r, cred,
				owner, claims)
			return oi, perm
		}

		switch ret.Mode {
		case objectlock.RetGovernance:
			govPerm := isPutRetentionAllowed(bucket, object, days,
				objRetention.RetainUntilDate.Time, objRetention.Mode,
				byPassSet, r, cred, owner, claims)
			if !byPassSet {
				if objRetention.Mode != objectlock.RetGovernance || objRetention.RetainUntilDate.Before((ret.RetainUntilDate.Time)) {
					return oi, ErrObjectLocked
				}
			}
			return oi, govPerm
		case objectlock.RetCompliance:
			if objRetention.Mode != objectlock.RetCompliance || objRetention.RetainUntilDate.Before((ret.RetainUntilDate.Time)) {
				return oi, ErrObjectLocked
			}
			compliancePerm := isPutRetentionAllowed(bucket, object,
				days, objRetention.RetainUntilDate.Time, objRetention.Mode,
				false, r, cred, owner, claims)
			return oi, compliancePerm
		}
		return oi, ErrNone
	}

	perm := isPutRetentionAllowed(bucket, object,
		days, objRetention.RetainUntilDate.Time,
		objRetention.Mode, byPassSet, r, cred, owner, claims)
	return oi, perm
}

func checkPutObjectLockAllowed(ctx context.Context, r *http.Request, bucket, object string, getObjectInfoFn GetObjectInfoFn, retentionPermErr, legalHoldPermErr APIErrorCode) (objectlock.RetMode, objectlock.RetentionDate, objectlock.ObjectLegalHold, APIErrorCode) {
	var mode objectlock.RetMode
	var retainDate objectlock.RetentionDate
	var legalHold objectlock.ObjectLegalHold

	retentionRequested := objectlock.IsObjectLockRetentionRequested(r.Header)
	legalHoldRequested := objectlock.IsObjectLockLegalHoldRequested(r.Header)

	retentionCfg, isWORMBucket := globalBucketObjectLockConfig.Get(bucket)
	if !isWORMBucket {
		if legalHoldRequested || retentionRequested {
			return mode, retainDate, legalHold, ErrInvalidBucketObjectLockConfiguration
		}
		return mode, retainDate, legalHold, ErrNone
	}

	var objExists bool
	opts, err := getOpts(ctx, r, bucket, object)
	if err != nil {
		return mode, retainDate, legalHold, toAPIErrorCode(ctx, err)
	}

	t, err := objectlock.UTCNowNTP()
	if err != nil {
		logger.LogIf(ctx, err)
		return mode, retainDate, legalHold, ErrObjectLocked
	}

	if objInfo, err := getObjectInfoFn(ctx, bucket, object, opts); err == nil {
		objExists = true
		r := objectlock.GetObjectRetentionMeta(objInfo.UserDefined)
		if r.Mode == objectlock.RetCompliance && r.RetainUntilDate.After(t) {
			return mode, retainDate, legalHold, ErrObjectLocked
		}
		mode = r.Mode
		retainDate = r.RetainUntilDate
		legalHold = objectlock.GetObjectLegalHoldMeta(objInfo.UserDefined)
		if legalHold.Status == objectlock.LegalHoldOn {
			return mode, retainDate, legalHold, ErrObjectLocked
		}
	}

	if legalHoldRequested {
		var lerr error
		if legalHold, lerr = objectlock.ParseObjectLockLegalHoldHeaders(r.Header); lerr != nil {
			return mode, retainDate, legalHold, toAPIErrorCode(ctx, err)
		}
	}

	if retentionRequested {
		legalHold, err := objectlock.ParseObjectLockLegalHoldHeaders(r.Header)
		if err != nil {
			return mode, retainDate, legalHold, toAPIErrorCode(ctx, err)
		}
		rMode, rDate, err := objectlock.ParseObjectLockRetentionHeaders(r.Header)
		if err != nil {
			return mode, retainDate, legalHold, toAPIErrorCode(ctx, err)
		}
		if objExists && retainDate.After(t) {
			return mode, retainDate, legalHold, ErrObjectLocked
		}
		if retentionPermErr != ErrNone {
			return mode, retainDate, legalHold, retentionPermErr
		}
		return rMode, rDate, legalHold, ErrNone
	}

	if !retentionRequested && isWORMBucket {
		if retentionPermErr != ErrNone {
			return mode, retainDate, legalHold, retentionPermErr
		}
		t, err := objectlock.UTCNowNTP()
		if err != nil {
			logger.LogIf(ctx, err)
			return mode, retainDate, legalHold, ErrObjectLocked
		}
		if objExists && retainDate.After(t) {
			return mode, retainDate, legalHold, ErrObjectLocked
		}
		if !legalHoldRequested && !retentionCfg.IsEmpty() {
			return retentionCfg.Mode, objectlock.RetentionDate{Time: t.Add(retentionCfg.Validity)}, legalHold, ErrNone
		}
		return "", objectlock.RetentionDate{}, legalHold, ErrNone
	}
	return mode, retainDate, legalHold, ErrNone
}

func initBucketObjectLockConfig(buckets []BucketInfo, objAPI ObjectLayer) error {
	for _, bucket := range buckets {
		ctx := logger.SetReqInfo(GlobalContext, &logger.ReqInfo{BucketName: bucket.Name})
		configFile := path.Join(bucketConfigPrefix, bucket.Name, bucketObjectLockEnabledConfigFile)
		bucketObjLockData, err := readConfig(ctx, objAPI, configFile)
		if err != nil {
			if errors.Is(err, errConfigNotFound) {
				continue
			}
			return err
		}

		if string(bucketObjLockData) != bucketObjectLockEnabledConfig {
			logger.LogIf(ctx, objectlock.ErrMalformedBucketObjectConfig)
			continue
		}

		configFile = path.Join(bucketConfigPrefix, bucket.Name, objectLockConfig)
		configData, err := readConfig(ctx, objAPI, configFile)
		if err != nil {
			if errors.Is(err, errConfigNotFound) {
				globalBucketObjectLockConfig.Set(bucket.Name, objectlock.Retention{})
				continue
			}
			return err
		}

		config, err := objectlock.ParseObjectLockConfig(bytes.NewReader(configData))
		if err != nil {
			return err
		}
		retention := objectlock.Retention{}
		if config.Rule != nil {
			retention = config.ToRetention()
		}
		globalBucketObjectLockConfig.Set(bucket.Name, retention)
	}
	return nil
}
