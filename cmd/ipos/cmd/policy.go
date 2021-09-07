package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/bucket/policy"
	"github.com/storeros/ipos/pkg/handlers"
	iposgopolicy "github.com/storeros/ipos/pkg/policy"
)

type PolicySys struct {
	sync.RWMutex
	bucketPolicyMap map[string]policy.Policy
}

func (sys *PolicySys) Set(bucketName string, policy policy.Policy) {
	sys.Lock()
	defer sys.Unlock()

	if policy.IsEmpty() {
		delete(sys.bucketPolicyMap, bucketName)
	} else {
		sys.bucketPolicyMap[bucketName] = policy
	}
}

func (sys *PolicySys) Remove(bucketName string) {
	sys.Lock()
	defer sys.Unlock()

	delete(sys.bucketPolicyMap, bucketName)
}

func (sys *PolicySys) IsAllowed(args policy.Args) bool {
	sys.RLock()
	defer sys.RUnlock()

	if p, found := sys.bucketPolicyMap[args.BucketName]; found {
		return p.IsAllowed(args)
	}

	return args.IsOwner
}

func (sys *PolicySys) load(buckets []BucketInfo, objAPI ObjectLayer) error {
	for _, bucket := range buckets {
		config, err := objAPI.GetBucketPolicy(GlobalContext, bucket.Name)
		if err != nil {
			if _, ok := err.(BucketPolicyNotFound); ok {
				sys.Remove(bucket.Name)
			}
			continue
		}
		if config.Version == "" {
			logger.Info("Found in-consistent bucket policies, Migrating them for Bucket: (%s)", bucket.Name)
			config.Version = policy.DefaultVersion

			if err = savePolicyConfig(GlobalContext, objAPI, bucket.Name, config); err != nil {
				logger.LogIf(GlobalContext, err)
				return err
			}
		}
		sys.Set(bucket.Name, *config)
	}
	return nil
}

func (sys *PolicySys) Init(buckets []BucketInfo, objAPI ObjectLayer) error {
	if objAPI == nil {
		return errInvalidArgument
	}

	return sys.load(buckets, objAPI)
}

func NewPolicySys() *PolicySys {
	return &PolicySys{
		bucketPolicyMap: make(map[string]policy.Policy),
	}
}

func getConditionValues(r *http.Request, lc string, username string, claims map[string]interface{}) map[string][]string {
	currTime := UTCNow()

	principalType := "Anonymous"
	if username != "" {
		principalType = "User"
	}

	args := map[string][]string{
		"CurrentTime":     {currTime.Format(time.RFC3339)},
		"EpochTime":       {strconv.FormatInt(currTime.Unix(), 10)},
		"SecureTransport": {strconv.FormatBool(r.TLS != nil)},
		"SourceIp":        {handlers.GetSourceIP(r)},
		"UserAgent":       {r.UserAgent()},
		"Referer":         {r.Referer()},
		"principaltype":   {principalType},
		"userid":          {username},
		"username":        {username},
	}

	if lc != "" {
		args["LocationConstraint"] = []string{lc}
	}

	cloneHeader := r.Header.Clone()

	for _, objLock := range []string{
		xhttp.AmzObjectLockMode,
		xhttp.AmzObjectLockLegalHold,
		xhttp.AmzObjectLockRetainUntilDate,
	} {
		if values, ok := cloneHeader[objLock]; ok {
			args[strings.TrimPrefix(objLock, "X-Amz-")] = values
		}
		cloneHeader.Del(objLock)
	}

	for key, values := range cloneHeader {
		if existingValues, found := args[key]; found {
			args[key] = append(existingValues, values...)
		} else {
			args[key] = values
		}
	}

	var cloneURLValues = url.Values{}
	for k, v := range r.URL.Query() {
		cloneURLValues[k] = v
	}

	for _, objLock := range []string{
		xhttp.AmzObjectLockMode,
		xhttp.AmzObjectLockLegalHold,
		xhttp.AmzObjectLockRetainUntilDate,
	} {
		if values, ok := cloneURLValues[objLock]; ok {
			args[strings.TrimPrefix(objLock, "X-Amz-")] = values
		}
		cloneURLValues.Del(objLock)
	}

	for key, values := range cloneURLValues {
		if existingValues, found := args[key]; found {
			args[key] = append(existingValues, values...)
		} else {
			args[key] = values
		}
	}

	for k, v := range claims {
		vStr, ok := v.(string)
		if ok {
			args[k] = []string{vStr}
		}
	}

	return args
}

func getPolicyConfig(objAPI ObjectLayer, bucketName string) (*policy.Policy, error) {
	configFile := path.Join(bucketConfigPrefix, bucketName, bucketPolicyConfig)

	configData, err := readConfig(GlobalContext, objAPI, configFile)
	if err != nil {
		if err == errConfigNotFound {
			err = BucketPolicyNotFound{Bucket: bucketName}
		}

		return nil, err
	}

	return policy.ParseConfig(bytes.NewReader(configData), bucketName)
}

func savePolicyConfig(ctx context.Context, objAPI ObjectLayer, bucketName string, bucketPolicy *policy.Policy) error {
	data, err := json.Marshal(bucketPolicy)
	if err != nil {
		return err
	}

	configFile := path.Join(bucketConfigPrefix, bucketName, bucketPolicyConfig)

	return saveConfig(ctx, objAPI, configFile, data)
}

func removePolicyConfig(ctx context.Context, objAPI ObjectLayer, bucketName string) error {
	configFile := path.Join(bucketConfigPrefix, bucketName, bucketPolicyConfig)

	if err := objAPI.DeleteObject(ctx, iposMetaBucket, configFile); err != nil {
		if _, ok := err.(ObjectNotFound); ok {
			return BucketPolicyNotFound{Bucket: bucketName}
		}

		return err
	}

	return nil
}

func PolicyToBucketAccessPolicy(bucketPolicy *policy.Policy) (*iposgopolicy.BucketAccessPolicy, error) {
	if bucketPolicy == nil {
		return &iposgopolicy.BucketAccessPolicy{Version: policy.DefaultVersion}, nil
	}

	data, err := json.Marshal(bucketPolicy)
	if err != nil {
		return nil, err
	}

	var policyInfo iposgopolicy.BucketAccessPolicy
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err = json.Unmarshal(data, &policyInfo); err != nil {
		return nil, err
	}

	return &policyInfo, nil
}

func BucketAccessPolicyToPolicy(policyInfo *iposgopolicy.BucketAccessPolicy) (*policy.Policy, error) {
	data, err := json.Marshal(policyInfo)
	if err != nil {
		return nil, err
	}

	var bucketPolicy policy.Policy
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err = json.Unmarshal(data, &bucketPolicy); err != nil {
		return nil, err
	}

	return &bucketPolicy, nil
}
