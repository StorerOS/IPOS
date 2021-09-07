package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/handlers"
)

const (
	kmBucket   = "BucketName"
	kmObject   = "ObjectName"
	kmObjects  = "Objects"
	kmPrefix   = "Prefix"
	kmMarker   = "Marker"
	kmUsername = "UserName"
	kmHostname = "HostName"
	kmPolicy   = "Policy"
)

type KeyValueMap map[string]string

func (km KeyValueMap) Bucket() string {
	return km[kmBucket]
}

func (km KeyValueMap) Object() string {
	return km[kmObject]
}

func (km KeyValueMap) Prefix() string {
	return km[kmPrefix]
}

func (km KeyValueMap) Username() string {
	return km[kmUsername]
}

func (km KeyValueMap) Hostname() string {
	return km[kmHostname]
}

func (km KeyValueMap) Policy() string {
	return km[kmPolicy]
}

func (km KeyValueMap) Objects() []string {
	var objects []string
	_ = json.Unmarshal([]byte(km[kmObjects]), &objects)
	return objects
}

func (km *KeyValueMap) SetBucket(bucket string) {
	(*km)[kmBucket] = bucket
}

func (km *KeyValueMap) SetPrefix(prefix string) {
	(*km)[kmPrefix] = prefix
}

func (km *KeyValueMap) SetObject(object string) {
	(*km)[kmObject] = object
}

func (km *KeyValueMap) SetMarker(marker string) {
	(*km)[kmMarker] = marker
}

func (km *KeyValueMap) SetPolicy(policy string) {
	(*km)[kmPolicy] = policy
}

func (km *KeyValueMap) SetExpiry(expiry int64) {
	(*km)[kmPolicy] = fmt.Sprintf("%d", expiry)
}

func (km *KeyValueMap) SetObjects(objects []string) {
	objsVal, err := json.Marshal(objects)
	if err != nil {
		objsVal = []byte("[]")
	}
	(*km)[kmObjects] = string(objsVal)
}

func (km *KeyValueMap) SetUsername(username string) {
	(*km)[kmUsername] = username
}

func (km *KeyValueMap) SetHostname(hostname string) {
	(*km)[kmHostname] = hostname
}

type ToKeyValuer interface {
	ToKeyValue() KeyValueMap
}

func (args *WebGenericArgs) ToKeyValue() KeyValueMap {
	return KeyValueMap{}
}

func (args *MakeBucketArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetBucket(args.BucketName)
	return km
}

func (args *RemoveBucketArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetBucket(args.BucketName)
	return km
}

func (args *ListObjectsArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetBucket(args.BucketName)
	km.SetPrefix(args.Prefix)
	km.SetMarker(args.Marker)
	return km
}

func (args *RemoveObjectArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetBucket(args.BucketName)
	km.SetObjects(args.Objects)
	return km
}

func (args *LoginArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetUsername(args.Username)
	return km
}

func (args *LoginSTSArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	return km
}

func (args *GetBucketPolicyArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetBucket(args.BucketName)
	km.SetPrefix(args.Prefix)
	return km
}

func (args *ListAllBucketPoliciesArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetBucket(args.BucketName)
	return km
}

func (args *SetBucketPolicyWebArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetBucket(args.BucketName)
	km.SetPrefix(args.Prefix)
	km.SetPolicy(args.Policy)
	return km
}

func (args *SetAuthArgs) ToKeyValue() KeyValueMap {
	return KeyValueMap{}
}

func (args *PresignedGetArgs) ToKeyValue() KeyValueMap {
	km := KeyValueMap{}
	km.SetHostname(args.HostName)
	km.SetBucket(args.BucketName)
	km.SetObject(args.ObjectName)
	km.SetExpiry(args.Expiry)
	return km
}

func newWebContext(r *http.Request, args ToKeyValuer, api string) context.Context {
	argsMap := args.ToKeyValue()
	bucket := argsMap.Bucket()
	object := argsMap.Object()
	prefix := argsMap.Prefix()

	if prefix != "" {
		object = prefix
	}
	reqInfo := &logger.ReqInfo{
		DeploymentID: globalDeploymentID,
		RemoteHost:   handlers.GetSourceIP(r),
		Host:         getHostName(r),
		UserAgent:    r.UserAgent(),
		API:          api,
		BucketName:   bucket,
		ObjectName:   object,
	}
	return logger.SetReqInfo(GlobalContext, reqInfo)
}
