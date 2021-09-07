package cmd

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2/json2"
	"github.com/klauspost/compress/zip"

	"github.com/storeros/ipos/browser"
	"github.com/storeros/ipos/cmd/ipos/crypto"
	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/auth"
	objectlock "github.com/storeros/ipos/pkg/bucket/object/lock"
	"github.com/storeros/ipos/pkg/bucket/policy"
	"github.com/storeros/ipos/pkg/hash"
	iampolicy "github.com/storeros/ipos/pkg/iam/policy"
	"github.com/storeros/ipos/pkg/ioutil"
	iposgopolicy "github.com/storeros/ipos/pkg/policy"
	"github.com/storeros/ipos/pkg/s3utils"
	"github.com/storeros/ipos/version"
)

type WebGenericArgs struct{}

type WebGenericRep struct {
	UIVersion string `json:"uiVersion"`
}

type ServerInfoRep struct {
	IPOSVersion    string
	IPOSMemory     string
	IPOSPlatform   string
	IPOSRuntime    string
	IPOSGlobalInfo map[string]interface{}
	IPOSUserInfo   map[string]interface{}
	UIVersion      string `json:"uiVersion"`
}

func (web *webAPIHandlers) ServerInfo(r *http.Request, args *WebGenericArgs, reply *ServerInfoRep) error {
	ctx := newWebContext(r, args, "WebServerInfo")
	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	platform := fmt.Sprintf("Host: %s | OS: %s | Arch: %s",
		host,
		runtime.GOOS,
		runtime.GOARCH)
	goruntime := fmt.Sprintf("Version: %s | CPUs: %d", runtime.Version(), runtime.NumCPU())

	reply.IPOSVersion = version.Version
	reply.IPOSGlobalInfo = getGlobalInfo()

	reply.IPOSUserInfo = map[string]interface{}{
		"isIAMUser": !owner,
	}

	if !owner {
		creds, ok := globalIAMSys.GetUser(claims.AccessKey)
		if ok && creds.SessionToken != "" {
			reply.IPOSUserInfo["isTempUser"] = true
		}
	}

	reply.IPOSPlatform = platform
	reply.IPOSRuntime = goruntime
	reply.UIVersion = browser.UIVersion
	return nil
}

type StorageInfoRep struct {
	StorageInfo StorageInfo `json:"storageInfo"`
	UIVersion   string      `json:"uiVersion"`
}

func (web *webAPIHandlers) StorageInfo(r *http.Request, args *WebGenericArgs, reply *StorageInfoRep) error {
	ctx := newWebContext(r, args, "WebStorageInfo")
	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}
	_, _, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}
	reply.StorageInfo = objectAPI.StorageInfo(ctx, false)
	reply.UIVersion = browser.UIVersion
	return nil
}

type MakeBucketArgs struct {
	BucketName string `json:"bucketName"`
}

func (web *webAPIHandlers) MakeBucket(r *http.Request, args *MakeBucketArgs, reply *WebGenericRep) error {
	ctx := newWebContext(r, args, "WebMakeBucket")
	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}
	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}

	if !globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     claims.AccessKey,
		Action:          iampolicy.CreateBucketAction,
		BucketName:      args.BucketName,
		ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
		IsOwner:         owner,
		Claims:          claims.Map(),
	}) {
		return toJSONError(ctx, errAccessDenied)
	}

	if isReservedOrInvalidBucket(args.BucketName, true) {
		return toJSONError(ctx, errInvalidBucketName)
	}

	if err := objectAPI.MakeBucketWithLocation(ctx, args.BucketName, globalServerRegion); err != nil {
		return toJSONError(ctx, err, args.BucketName)
	}

	reply.UIVersion = browser.UIVersion
	return nil
}

type RemoveBucketArgs struct {
	BucketName string `json:"bucketName"`
}

func (web *webAPIHandlers) DeleteBucket(r *http.Request, args *RemoveBucketArgs, reply *WebGenericRep) error {
	ctx := newWebContext(r, args, "WebDeleteBucket")
	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}
	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}

	if !globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     claims.AccessKey,
		Action:          iampolicy.DeleteBucketAction,
		BucketName:      args.BucketName,
		ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
		IsOwner:         owner,
		Claims:          claims.Map(),
	}) {
		return toJSONError(ctx, errAccessDenied)
	}

	if isReservedOrInvalidBucket(args.BucketName, false) {
		return toJSONError(ctx, errInvalidBucketName)
	}

	reply.UIVersion = browser.UIVersion

	deleteBucket := objectAPI.DeleteBucket

	if err := deleteBucket(ctx, args.BucketName, false); err != nil {
		return toJSONError(ctx, err, args.BucketName)
	}

	return nil
}

type ListBucketsRep struct {
	Buckets   []WebBucketInfo `json:"buckets"`
	UIVersion string          `json:"uiVersion"`
}

type WebBucketInfo struct {
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creationDate"`
}

func (web *webAPIHandlers) ListBuckets(r *http.Request, args *WebGenericArgs, reply *ListBucketsRep) error {
	ctx := newWebContext(r, args, "WebListBuckets")
	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}
	listBuckets := objectAPI.ListBuckets

	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}

	r.Header.Set("prefix", "")

	r.Header.Set("delimiter", SlashSeparator)

	buckets, err := listBuckets(ctx)
	if err != nil {
		return toJSONError(ctx, err)
	}
	for _, bucket := range buckets {
		if globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.ListAllMyBucketsAction,
			BucketName:      bucket.Name,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			ObjectName:      "",
			Claims:          claims.Map(),
		}) {
			reply.Buckets = append(reply.Buckets, WebBucketInfo{
				Name:         bucket.Name,
				CreationDate: bucket.Created,
			})
		}
	}

	reply.UIVersion = browser.UIVersion
	return nil
}

type ListObjectsArgs struct {
	BucketName string `json:"bucketName"`
	Prefix     string `json:"prefix"`
	Marker     string `json:"marker"`
}

type ListObjectsRep struct {
	Objects   []WebObjectInfo `json:"objects"`
	Writable  bool            `json:"writable"`
	UIVersion string          `json:"uiVersion"`
}

type WebObjectInfo struct {
	Key          string    `json:"name"`
	LastModified time.Time `json:"lastModified"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType"`
}

func (web *webAPIHandlers) ListObjects(r *http.Request, args *ListObjectsArgs, reply *ListObjectsRep) error {
	ctx := newWebContext(r, args, "WebListObjects")
	reply.UIVersion = browser.UIVersion
	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}

	listObjects := objectAPI.ListObjects

	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		if authErr == errNoAuthToken {
			r.Header.Set("prefix", args.Prefix)

			r.Header.Set("delimiter", SlashSeparator)

			readable := globalPolicySys.IsAllowed(policy.Args{
				Action:          policy.ListBucketAction,
				BucketName:      args.BucketName,
				ConditionValues: getConditionValues(r, "", "", nil),
				IsOwner:         false,
			})

			writable := globalPolicySys.IsAllowed(policy.Args{
				Action:          policy.PutObjectAction,
				BucketName:      args.BucketName,
				ConditionValues: getConditionValues(r, "", "", nil),
				IsOwner:         false,
				ObjectName:      args.Prefix + SlashSeparator,
			})

			reply.Writable = writable
			if !readable {
				if !writable {
					return errAccessDenied
				}
				return nil
			}
		} else {
			return toJSONError(ctx, authErr)
		}
	}

	if authErr == nil {
		r.Header.Set("prefix", args.Prefix)

		r.Header.Set("delimiter", SlashSeparator)

		readable := globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.ListBucketAction,
			BucketName:      args.BucketName,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			Claims:          claims.Map(),
		})

		writable := globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.PutObjectAction,
			BucketName:      args.BucketName,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			ObjectName:      args.Prefix + SlashSeparator,
			Claims:          claims.Map(),
		})

		reply.Writable = writable
		if !readable {
			if !writable {
				return errAccessDenied
			}
			return nil
		}
	}

	if isReservedOrInvalidBucket(args.BucketName, false) {
		return toJSONError(ctx, errInvalidBucketName)
	}

	nextMarker := ""
	for {
		lo, err := listObjects(ctx, args.BucketName, args.Prefix, nextMarker, SlashSeparator, maxObjectList)
		if err != nil {
			return &json2.Error{Message: err.Error()}
		}
		for i := range lo.Objects {
			if crypto.IsEncrypted(lo.Objects[i].UserDefined) {
				lo.Objects[i].Size, err = lo.Objects[i].DecryptedSize()
				if err != nil {
					return toJSONError(ctx, err)
				}
			} else if lo.Objects[i].IsCompressed() {
				actualSize := lo.Objects[i].GetActualSize()
				if actualSize < 0 {
					return toJSONError(ctx, errInvalidDecompressedSize)
				}
				lo.Objects[i].Size = actualSize
			}
		}

		for _, obj := range lo.Objects {
			reply.Objects = append(reply.Objects, WebObjectInfo{
				Key:          obj.Name,
				LastModified: obj.ModTime,
				Size:         obj.Size,
				ContentType:  obj.ContentType,
			})
		}
		for _, prefix := range lo.Prefixes {
			reply.Objects = append(reply.Objects, WebObjectInfo{
				Key: prefix,
			})
		}

		nextMarker = lo.NextMarker

		if !lo.IsTruncated {
			return nil
		}
	}
}

type RemoveObjectArgs struct {
	Objects    []string `json:"objects"`
	BucketName string   `json:"bucketname"`
}

func (web *webAPIHandlers) RemoveObject(r *http.Request, args *RemoveObjectArgs, reply *WebGenericRep) error {
	ctx := newWebContext(r, args, "WebRemoveObject")
	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}

	getObjectInfo := objectAPI.GetObjectInfo

	deleteObjects := objectAPI.DeleteObjects

	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		if authErr == errNoAuthToken {
			for _, object := range args.Objects {
				if !globalPolicySys.IsAllowed(policy.Args{
					Action:          policy.DeleteObjectAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", "", nil),
					IsOwner:         false,
					ObjectName:      object,
				}) {
					return toJSONError(ctx, errAuthentication)
				}
			}
		} else {
			return toJSONError(ctx, authErr)
		}
	}

	if args.BucketName == "" || len(args.Objects) == 0 {
		return toJSONError(ctx, errInvalidArgument)
	}

	if isReservedOrInvalidBucket(args.BucketName, false) {
		return toJSONError(ctx, errInvalidBucketName)
	}

	reply.UIVersion = browser.UIVersion

	var err error
next:
	for _, objectName := range args.Objects {
		if !HasSuffix(objectName, SlashSeparator) && objectName != "" {
			govBypassPerms := ErrAccessDenied
			if authErr != errNoAuthToken {
				if !globalIAMSys.IsAllowed(iampolicy.Args{
					AccountName:     claims.AccessKey,
					Action:          iampolicy.DeleteObjectAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
					IsOwner:         owner,
					ObjectName:      objectName,
					Claims:          claims.Map(),
				}) {
					return toJSONError(ctx, errAccessDenied)
				}
				if globalIAMSys.IsAllowed(iampolicy.Args{
					AccountName:     claims.AccessKey,
					Action:          iampolicy.BypassGovernanceRetentionAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
					IsOwner:         owner,
					ObjectName:      objectName,
					Claims:          claims.Map(),
				}) {
					govBypassPerms = ErrNone
				}
				if globalIAMSys.IsAllowed(iampolicy.Args{
					AccountName:     claims.AccessKey,
					Action:          iampolicy.GetBucketObjectLockConfigurationAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
					IsOwner:         owner,
					ObjectName:      objectName,
					Claims:          claims.Map(),
				}) {
					govBypassPerms = ErrNone
				}
			}
			if authErr == errNoAuthToken {
				if !globalPolicySys.IsAllowed(policy.Args{
					Action:          policy.DeleteObjectAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", "", nil),
					IsOwner:         false,
					ObjectName:      objectName,
				}) {
					return toJSONError(ctx, errAccessDenied)
				}

				if globalPolicySys.IsAllowed(policy.Args{
					Action:          policy.BypassGovernanceRetentionAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", "", nil),
					IsOwner:         false,
					ObjectName:      objectName,
				}) {
					govBypassPerms = ErrNone
				}

				if globalPolicySys.IsAllowed(policy.Args{
					Action:          policy.GetBucketObjectLockConfigurationAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", "", nil),
					IsOwner:         false,
					ObjectName:      objectName,
				}) {
					govBypassPerms = ErrNone
				}
			}
			if govBypassPerms != ErrNone {
				return toJSONError(ctx, errAccessDenied)
			}

			apiErr := ErrNone
			if _, ok := globalBucketObjectLockConfig.Get(args.BucketName); ok && (apiErr == ErrNone) {
				apiErr = enforceRetentionBypassForDeleteWeb(ctx, r, args.BucketName, objectName, getObjectInfo)
				if apiErr != ErrNone && apiErr != ErrNoSuchKey {
					return toJSONError(ctx, errAccessDenied)
				}
			}
			if apiErr == ErrNone {
				if err = deleteObject(ctx, objectAPI, args.BucketName, objectName, r); err != nil {
					break next
				}
			}
			continue
		}

		if authErr == errNoAuthToken {
			if !globalPolicySys.IsAllowed(policy.Args{
				Action:          iampolicy.DeleteObjectAction,
				BucketName:      args.BucketName,
				ConditionValues: getConditionValues(r, "", "", nil),
				IsOwner:         false,
				ObjectName:      objectName,
			}) {
				return toJSONError(ctx, errAccessDenied)
			}
		} else {
			if !globalIAMSys.IsAllowed(iampolicy.Args{
				AccountName:     claims.AccessKey,
				Action:          iampolicy.DeleteObjectAction,
				BucketName:      args.BucketName,
				ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
				IsOwner:         owner,
				ObjectName:      objectName,
				Claims:          claims.Map(),
			}) {
				return toJSONError(ctx, errAccessDenied)
			}
		}

		objInfoCh := make(chan ObjectInfo)

		if err = objectAPI.Walk(ctx, args.BucketName, objectName, objInfoCh); err != nil {
			break next
		}

		for {
			var objects []string
			for obj := range objInfoCh {
				if len(objects) == maxObjectList {
					break
				}
				objects = append(objects, obj.Name)
			}

			if len(objects) == 0 {
				break next
			}

			_, err = deleteObjects(ctx, args.BucketName, objects)
			if err != nil {
				logger.LogIf(ctx, err)
				break next
			}
		}
	}

	if err != nil && !isErrObjectNotFound(err) {
		return toJSONError(ctx, err, args.BucketName, "")
	}

	return nil
}

type LoginArgs struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

type LoginRep struct {
	Token     string `json:"token"`
	UIVersion string `json:"uiVersion"`
}

func (web *webAPIHandlers) Login(r *http.Request, args *LoginArgs, reply *LoginRep) error {
	ctx := newWebContext(r, args, "WebLogin")
	token, err := authenticateWeb(args.Username, args.Password)
	if err != nil {
		return toJSONError(ctx, err)
	}

	reply.Token = token
	reply.UIVersion = browser.UIVersion
	return nil
}

type GenerateAuthReply struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	UIVersion string `json:"uiVersion"`
}

func (web webAPIHandlers) GenerateAuth(r *http.Request, args *WebGenericArgs, reply *GenerateAuthReply) error {
	ctx := newWebContext(r, args, "WebGenerateAuth")
	_, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}
	if !owner {
		return toJSONError(ctx, errAccessDenied)
	}
	cred, err := auth.GetNewCredentials()
	if err != nil {
		return toJSONError(ctx, err)
	}
	reply.AccessKey = cred.AccessKey
	reply.SecretKey = cred.SecretKey
	reply.UIVersion = browser.UIVersion
	return nil
}

type SetAuthArgs struct {
	CurrentAccessKey string `json:"currentAccessKey"`
	CurrentSecretKey string `json:"currentSecretKey"`
	NewAccessKey     string `json:"newAccessKey"`
	NewSecretKey     string `json:"newSecretKey"`
}

type SetAuthReply struct {
	Token       string            `json:"token"`
	UIVersion   string            `json:"uiVersion"`
	PeerErrMsgs map[string]string `json:"peerErrMsgs"`
}

func (web *webAPIHandlers) SetAuth(r *http.Request, args *SetAuthArgs, reply *SetAuthReply) error {
	ctx := newWebContext(r, args, "WebSetAuth")
	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}

	if owner {
		return toJSONError(ctx, errChangeCredNotAllowed)
	}

	prevCred, ok := globalIAMSys.GetUser(claims.AccessKey)
	if !ok {
		return errInvalidAccessKeyID
	}

	if prevCred.SecretKey != args.CurrentSecretKey {
		return errIncorrectCreds
	}

	creds, err := auth.CreateCredentials(claims.AccessKey, args.NewSecretKey)
	if err != nil {
		return toJSONError(ctx, err)
	}

	err = globalIAMSys.SetUserSecretKey(creds.AccessKey, creds.SecretKey)
	if err != nil {
		return toJSONError(ctx, err)
	}

	reply.Token, err = authenticateWeb(creds.AccessKey, creds.SecretKey)
	if err != nil {
		return toJSONError(ctx, err)
	}

	reply.UIVersion = browser.UIVersion

	return nil
}

type URLTokenReply struct {
	Token     string `json:"token"`
	UIVersion string `json:"uiVersion"`
}

func (web *webAPIHandlers) CreateURLToken(r *http.Request, args *WebGenericArgs, reply *URLTokenReply) error {
	ctx := newWebContext(r, args, "WebCreateURLToken")
	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}

	creds := globalActiveCred
	if !owner {
		var ok bool
		creds, ok = globalIAMSys.GetUser(claims.AccessKey)
		if !ok {
			return toJSONError(ctx, errInvalidAccessKeyID)
		}
	}

	if creds.SessionToken != "" {
		reply.Token = creds.SessionToken
	} else {
		token, err := authenticateURL(creds.AccessKey, creds.SecretKey)
		if err != nil {
			return toJSONError(ctx, err)
		}
		reply.Token = token
	}

	reply.UIVersion = browser.UIVersion
	return nil
}

func (web *webAPIHandlers) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "WebUpload")

	defer logger.AuditLog(w, r, "WebUpload", mustGetClaimsFromToken(r))

	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		writeWebErrorResponse(w, errServerNotInitialized)
		return
	}
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object, err := url.PathUnescape(vars["object"])
	if err != nil {
		writeWebErrorResponse(w, err)
		return
	}

	retPerms := ErrAccessDenied
	holdPerms := ErrAccessDenied

	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		if authErr == errNoAuthToken {
			if !globalPolicySys.IsAllowed(policy.Args{
				Action:          policy.PutObjectAction,
				BucketName:      bucket,
				ConditionValues: getConditionValues(r, "", "", nil),
				IsOwner:         false,
				ObjectName:      object,
			}) {
				writeWebErrorResponse(w, errAuthentication)
				return
			}
		} else {
			writeWebErrorResponse(w, authErr)
			return
		}
	}

	if authErr == nil {
		if !globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.PutObjectAction,
			BucketName:      bucket,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			ObjectName:      object,
			Claims:          claims.Map(),
		}) {
			writeWebErrorResponse(w, errAuthentication)
			return
		}
		if globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.PutObjectRetentionAction,
			BucketName:      bucket,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			ObjectName:      object,
			Claims:          claims.Map(),
		}) {
			retPerms = ErrNone
		}
		if globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.PutObjectLegalHoldAction,
			BucketName:      bucket,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			ObjectName:      object,
			Claims:          claims.Map(),
		}) {
			holdPerms = ErrNone
		}
	}

	if isReservedOrInvalidBucket(bucket, false) {
		writeWebErrorResponse(w, errInvalidBucketName)
		return
	}

	size := r.ContentLength
	if size < 0 {
		writeWebErrorResponse(w, errSizeUnspecified)
		return
	}

	metadata, err := extractMetadata(ctx, r)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	var pReader *PutObjReader
	var reader io.Reader = r.Body
	actualSize := size

	hashReader, err := hash.NewReader(reader, size, "", "", actualSize, globalCLIContext.StrictS3Compat)
	if err != nil {
		writeWebErrorResponse(w, err)
		return
	}
	pReader = NewPutObjReader(hashReader, nil, nil)
	var opts ObjectOptions
	opts, err = putOpts(ctx, r, bucket, object, metadata)
	if err != nil {
		writeErrorResponseHeadersOnly(w, toAPIError(ctx, err))
		return
	}
	if objectAPI.IsEncryptionSupported() {
		if crypto.IsRequested(r.Header) && !HasSuffix(object, SlashSeparator) {
			rawReader := hashReader
			var objectEncryptionKey crypto.ObjectKey
			reader, objectEncryptionKey, err = EncryptRequest(hashReader, r, bucket, object, metadata)
			if err != nil {
				writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
				return
			}
			info := ObjectInfo{Size: size}
			hashReader, err = hash.NewReader(reader, info.EncryptedSize(), "", "", size, globalCLIContext.StrictS3Compat)
			if err != nil {
				writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
				return
			}
			pReader = NewPutObjReader(rawReader, hashReader, &objectEncryptionKey)
		}
	}

	crypto.RemoveSensitiveEntries(metadata)

	retentionRequested := objectlock.IsObjectLockRetentionRequested(r.Header)
	legalHoldRequested := objectlock.IsObjectLockLegalHoldRequested(r.Header)

	putObject := objectAPI.PutObject
	getObjectInfo := objectAPI.GetObjectInfo

	if retentionRequested || legalHoldRequested {
		retentionMode, retentionDate, legalHold, s3Err := checkPutObjectLockAllowed(ctx, r, bucket, object, getObjectInfo, retPerms, holdPerms)
		if s3Err == ErrNone && retentionMode != "" {
			opts.UserDefined[xhttp.AmzObjectLockMode] = string(retentionMode)
			opts.UserDefined[xhttp.AmzObjectLockRetainUntilDate] = retentionDate.UTC().Format(time.RFC3339)
		}
		if s3Err == ErrNone && legalHold.Status != "" {
			opts.UserDefined[xhttp.AmzObjectLockLegalHold] = string(legalHold.Status)
		}
		if s3Err != ErrNone {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Err), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	objInfo, err := putObject(GlobalContext, bucket, object, pReader, opts)
	if err != nil {
		writeWebErrorResponse(w, err)
		return
	}
	if objectAPI.IsEncryptionSupported() {
		if crypto.IsEncrypted(objInfo.UserDefined) {
			switch {
			case crypto.S3.IsEncrypted(objInfo.UserDefined):
				w.Header().Set(crypto.SSEHeader, crypto.SSEAlgorithmAES256)
			case crypto.SSEC.IsRequested(r.Header):
				w.Header().Set(crypto.SSECAlgorithm, r.Header.Get(crypto.SSECAlgorithm))
				w.Header().Set(crypto.SSECKeyMD5, r.Header.Get(crypto.SSECKeyMD5))
			}
		}
	}
}

func (web *webAPIHandlers) Download(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "WebDownload")

	defer logger.AuditLog(w, r, "WebDownload", mustGetClaimsFromToken(r))

	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		writeWebErrorResponse(w, errServerNotInitialized)
		return
	}

	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object, err := url.PathUnescape(vars["object"])
	if err != nil {
		writeWebErrorResponse(w, err)
		return
	}
	token := r.URL.Query().Get("token")

	getRetPerms := ErrAccessDenied
	legalHoldPerms := ErrAccessDenied

	claims, owner, authErr := webTokenAuthenticate(token)
	if authErr != nil {
		if authErr == errNoAuthToken {
			if !globalPolicySys.IsAllowed(policy.Args{
				Action:          policy.GetObjectAction,
				BucketName:      bucket,
				ConditionValues: getConditionValues(r, "", "", nil),
				IsOwner:         false,
				ObjectName:      object,
			}) {
				writeWebErrorResponse(w, errAuthentication)
				return
			}
			if globalPolicySys.IsAllowed(policy.Args{
				Action:          policy.GetObjectRetentionAction,
				BucketName:      bucket,
				ConditionValues: getConditionValues(r, "", "", nil),
				IsOwner:         false,
				ObjectName:      object,
			}) {
				getRetPerms = ErrNone
			}
			if globalPolicySys.IsAllowed(policy.Args{
				Action:          policy.GetObjectLegalHoldAction,
				BucketName:      bucket,
				ConditionValues: getConditionValues(r, "", "", nil),
				IsOwner:         false,
				ObjectName:      object,
			}) {
				legalHoldPerms = ErrNone
			}
		} else {
			writeWebErrorResponse(w, authErr)
			return
		}
	}

	if authErr == nil {
		if !globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.GetObjectAction,
			BucketName:      bucket,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			ObjectName:      object,
			Claims:          claims.Map(),
		}) {
			writeWebErrorResponse(w, errAuthentication)
			return
		}
		if globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.GetObjectRetentionAction,
			BucketName:      bucket,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			ObjectName:      object,
			Claims:          claims.Map(),
		}) {
			getRetPerms = ErrNone
		}
		if globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     claims.AccessKey,
			Action:          iampolicy.GetObjectLegalHoldAction,
			BucketName:      bucket,
			ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
			IsOwner:         owner,
			ObjectName:      object,
			Claims:          claims.Map(),
		}) {
			legalHoldPerms = ErrNone
		}
	}

	if isReservedOrInvalidBucket(bucket, false) {
		writeWebErrorResponse(w, errInvalidBucketName)
		return
	}

	getObjectNInfo := objectAPI.GetObjectNInfo

	var opts ObjectOptions
	gr, err := getObjectNInfo(ctx, bucket, object, nil, r.Header, readLock, opts)
	if err != nil {
		writeWebErrorResponse(w, err)
		return
	}
	defer gr.Close()

	objInfo := gr.ObjInfo

	objInfo.UserDefined = objectlock.FilterObjectLockMetadata(objInfo.UserDefined, getRetPerms != ErrNone, legalHoldPerms != ErrNone)

	if objectAPI.IsEncryptionSupported() {
		if _, err = DecryptObjectInfo(&objInfo, r.Header); err != nil {
			writeWebErrorResponse(w, err)
			return
		}
	}

	if objectAPI.IsEncryptionSupported() {
		if crypto.IsEncrypted(objInfo.UserDefined) {
			switch {
			case crypto.S3.IsEncrypted(objInfo.UserDefined):
				w.Header().Set(crypto.SSEHeader, crypto.SSEAlgorithmAES256)
			case crypto.SSEC.IsEncrypted(objInfo.UserDefined):
				w.Header().Set(crypto.SSECAlgorithm, r.Header.Get(crypto.SSECAlgorithm))
				w.Header().Set(crypto.SSECKeyMD5, r.Header.Get(crypto.SSECKeyMD5))
			}
		}
	}

	if err = setObjectHeaders(w, objInfo, nil); err != nil {
		writeWebErrorResponse(w, err)
		return
	}

	w.Header().Set(xhttp.ContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", path.Base(objInfo.Name)))

	setHeadGetRespHeaders(w, r.URL.Query())

	httpWriter := ioutil.WriteOnClose(w)

	if _, err = io.Copy(httpWriter, gr); err != nil {
		if !httpWriter.HasWritten() {
			writeWebErrorResponse(w, err)
		}
		return
	}

	if err = httpWriter.Close(); err != nil {
		if !httpWriter.HasWritten() {
			writeWebErrorResponse(w, err)
			return
		}
	}
}

type DownloadZipArgs struct {
	Objects    []string `json:"objects"`
	Prefix     string   `json:"prefix"`
	BucketName string   `json:"bucketname"`
}

func (web *webAPIHandlers) DownloadZip(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "WebDownloadZip")
	defer logger.AuditLog(w, r, "WebDownloadZip", mustGetClaimsFromToken(r))

	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		writeWebErrorResponse(w, errServerNotInitialized)
		return
	}

	var args DownloadZipArgs
	tenKB := 10 * 1024
	decodeErr := json.NewDecoder(io.LimitReader(r.Body, int64(tenKB))).Decode(&args)
	if decodeErr != nil {
		writeWebErrorResponse(w, decodeErr)
		return
	}
	token := r.URL.Query().Get("token")
	claims, owner, authErr := webTokenAuthenticate(token)
	var getRetPerms []APIErrorCode
	var legalHoldPerms []APIErrorCode

	if authErr != nil {
		if authErr == errNoAuthToken {
			for _, object := range args.Objects {
				if !globalPolicySys.IsAllowed(policy.Args{
					Action:          policy.GetObjectAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", "", nil),
					IsOwner:         false,
					ObjectName:      pathJoin(args.Prefix, object),
				}) {
					writeWebErrorResponse(w, errAuthentication)
					return
				}
				retentionPerm := ErrAccessDenied
				if globalPolicySys.IsAllowed(policy.Args{
					Action:          policy.GetObjectRetentionAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", "", nil),
					IsOwner:         false,
					ObjectName:      pathJoin(args.Prefix, object),
				}) {
					retentionPerm = ErrNone
				}
				getRetPerms = append(getRetPerms, retentionPerm)

				legalHoldPerm := ErrAccessDenied
				if globalPolicySys.IsAllowed(policy.Args{
					Action:          policy.GetObjectLegalHoldAction,
					BucketName:      args.BucketName,
					ConditionValues: getConditionValues(r, "", "", nil),
					IsOwner:         false,
					ObjectName:      pathJoin(args.Prefix, object),
				}) {
					legalHoldPerm = ErrNone
				}
				legalHoldPerms = append(legalHoldPerms, legalHoldPerm)
			}
		} else {
			writeWebErrorResponse(w, authErr)
			return
		}
	}

	if authErr == nil {
		for _, object := range args.Objects {
			if !globalIAMSys.IsAllowed(iampolicy.Args{
				AccountName:     claims.AccessKey,
				Action:          iampolicy.GetObjectAction,
				BucketName:      args.BucketName,
				ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
				IsOwner:         owner,
				ObjectName:      pathJoin(args.Prefix, object),
				Claims:          claims.Map(),
			}) {
				writeWebErrorResponse(w, errAuthentication)
				return
			}
			retentionPerm := ErrAccessDenied
			if globalIAMSys.IsAllowed(iampolicy.Args{
				AccountName:     claims.AccessKey,
				Action:          iampolicy.GetObjectRetentionAction,
				BucketName:      args.BucketName,
				ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
				IsOwner:         owner,
				ObjectName:      pathJoin(args.Prefix, object),
				Claims:          claims.Map(),
			}) {
				retentionPerm = ErrNone
			}
			getRetPerms = append(getRetPerms, retentionPerm)

			legalHoldPerm := ErrAccessDenied
			if globalIAMSys.IsAllowed(iampolicy.Args{
				AccountName:     claims.AccessKey,
				Action:          iampolicy.GetObjectLegalHoldAction,
				BucketName:      args.BucketName,
				ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
				IsOwner:         owner,
				ObjectName:      pathJoin(args.Prefix, object),
				Claims:          claims.Map(),
			}) {
				legalHoldPerm = ErrNone
			}
			legalHoldPerms = append(legalHoldPerms, legalHoldPerm)
		}
	}

	if isReservedOrInvalidBucket(args.BucketName, false) {
		writeWebErrorResponse(w, errInvalidBucketName)
		return
	}

	getObjectNInfo := objectAPI.GetObjectNInfo

	archive := zip.NewWriter(w)
	defer archive.Close()

	for i, object := range args.Objects {
		zipit := func(objectName string) error {
			var opts ObjectOptions
			gr, err := getObjectNInfo(ctx, args.BucketName, objectName, nil, r.Header, readLock, opts)
			if err != nil {
				return err
			}
			defer gr.Close()

			info := gr.ObjInfo
			info.UserDefined = objectlock.FilterObjectLockMetadata(info.UserDefined, getRetPerms[i] != ErrNone, legalHoldPerms[i] != ErrNone)

			if info.IsCompressed() {
				info.Size = info.GetActualSize()
			}
			header := &zip.FileHeader{
				Name:     strings.TrimPrefix(objectName, args.Prefix),
				Method:   zip.Deflate,
				Flags:    1 << 11,
				Modified: info.ModTime,
			}
			if hasStringSuffixInSlice(info.Name, standardExcludeCompressExtensions) || hasPattern(standardExcludeCompressContentTypes, info.ContentType) {
				header.Method = zip.Store
			}
			writer, err := archive.CreateHeader(header)
			if err != nil {
				writeWebErrorResponse(w, err)
				return err
			}
			httpWriter := ioutil.WriteOnClose(writer)

			if _, err = io.Copy(httpWriter, gr); err != nil {
				httpWriter.Close()
				if !httpWriter.HasWritten() {
					writeWebErrorResponse(w, err)
				}
				return err
			}

			if err = httpWriter.Close(); err != nil {
				if !httpWriter.HasWritten() {
					writeWebErrorResponse(w, err)
					return err
				}
			}

			return nil
		}

		if !HasSuffix(object, SlashSeparator) {
			err := zipit(pathJoin(args.Prefix, object))
			if err != nil {
				logger.LogIf(ctx, err)
				return
			}
			continue
		}

		objInfoCh := make(chan ObjectInfo)

		if err := objectAPI.Walk(ctx, args.BucketName, pathJoin(args.Prefix, object), objInfoCh); err != nil {
			logger.LogIf(ctx, err)
			continue
		}

		for obj := range objInfoCh {
			if err := zipit(obj.Name); err != nil {
				logger.LogIf(ctx, err)
				continue
			}
		}
	}
}

type GetBucketPolicyArgs struct {
	BucketName string `json:"bucketName"`
	Prefix     string `json:"prefix"`
}

type GetBucketPolicyRep struct {
	UIVersion string                    `json:"uiVersion"`
	Policy    iposgopolicy.BucketPolicy `json:"policy"`
}

func (web *webAPIHandlers) GetBucketPolicy(r *http.Request, args *GetBucketPolicyArgs, reply *GetBucketPolicyRep) error {
	ctx := newWebContext(r, args, "WebGetBucketPolicy")
	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}

	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}

	if !globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     claims.AccessKey,
		Action:          iampolicy.GetBucketPolicyAction,
		BucketName:      args.BucketName,
		ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
		IsOwner:         owner,
		Claims:          claims.Map(),
	}) {
		return toJSONError(ctx, errAccessDenied)
	}

	if isReservedOrInvalidBucket(args.BucketName, false) {
		return toJSONError(ctx, errInvalidBucketName)
	}

	var policyInfo = &iposgopolicy.BucketAccessPolicy{Version: "2012-10-17"}

	bucketPolicy, err := objectAPI.GetBucketPolicy(ctx, args.BucketName)
	if err != nil {
		if _, ok := err.(BucketPolicyNotFound); !ok {
			return toJSONError(ctx, err, args.BucketName)
		}
		return err
	}

	policyInfo, err = PolicyToBucketAccessPolicy(bucketPolicy)
	if err != nil {
		return toJSONError(ctx, err, args.BucketName)
	}

	reply.UIVersion = browser.UIVersion
	reply.Policy = iposgopolicy.GetPolicy(policyInfo.Statements, args.BucketName, args.Prefix)

	return nil
}

type ListAllBucketPoliciesArgs struct {
	BucketName string `json:"bucketName"`
}

type BucketAccessPolicy struct {
	Bucket string                    `json:"bucket"`
	Prefix string                    `json:"prefix"`
	Policy iposgopolicy.BucketPolicy `json:"policy"`
}

type ListAllBucketPoliciesRep struct {
	UIVersion string               `json:"uiVersion"`
	Policies  []BucketAccessPolicy `json:"policies"`
}

func (web *webAPIHandlers) ListAllBucketPolicies(r *http.Request, args *ListAllBucketPoliciesArgs, reply *ListAllBucketPoliciesRep) error {
	ctx := newWebContext(r, args, "WebListAllBucketPolicies")
	objectAPI := web.ObjectAPI()
	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}

	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}

	if !globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     claims.AccessKey,
		Action:          iampolicy.GetBucketPolicyAction,
		BucketName:      args.BucketName,
		ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
		IsOwner:         owner,
		Claims:          claims.Map(),
	}) {
		return toJSONError(ctx, errAccessDenied)
	}

	if isReservedOrInvalidBucket(args.BucketName, false) {
		return toJSONError(ctx, errInvalidBucketName)
	}

	var policyInfo = new(iposgopolicy.BucketAccessPolicy)

	bucketPolicy, err := objectAPI.GetBucketPolicy(ctx, args.BucketName)
	if err != nil {
		if _, ok := err.(BucketPolicyNotFound); !ok {
			return toJSONError(ctx, err, args.BucketName)
		}
	}
	policyInfo, err = PolicyToBucketAccessPolicy(bucketPolicy)
	if err != nil {
		return toJSONError(ctx, err, args.BucketName)
	}

	reply.UIVersion = browser.UIVersion
	for prefix, policy := range iposgopolicy.GetPolicies(policyInfo.Statements, args.BucketName, "") {
		bucketName, objectPrefix := path2BucketObject(prefix)
		objectPrefix = strings.TrimSuffix(objectPrefix, "*")
		reply.Policies = append(reply.Policies, BucketAccessPolicy{
			Bucket: bucketName,
			Prefix: objectPrefix,
			Policy: policy,
		})
	}

	return nil
}

type SetBucketPolicyWebArgs struct {
	BucketName string `json:"bucketName"`
	Prefix     string `json:"prefix"`
	Policy     string `json:"policy"`
}

func (web *webAPIHandlers) SetBucketPolicy(r *http.Request, args *SetBucketPolicyWebArgs, reply *WebGenericRep) error {
	ctx := newWebContext(r, args, "WebSetBucketPolicy")
	objectAPI := web.ObjectAPI()
	reply.UIVersion = browser.UIVersion

	if objectAPI == nil {
		return toJSONError(ctx, errServerNotInitialized)
	}

	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}

	if !globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     claims.AccessKey,
		Action:          iampolicy.PutBucketPolicyAction,
		BucketName:      args.BucketName,
		ConditionValues: getConditionValues(r, "", claims.AccessKey, claims.Map()),
		IsOwner:         owner,
		Claims:          claims.Map(),
	}) {
		return toJSONError(ctx, errAccessDenied)
	}

	if isReservedOrInvalidBucket(args.BucketName, false) {
		return toJSONError(ctx, errInvalidBucketName)
	}

	policyType := iposgopolicy.BucketPolicy(args.Policy)
	if !policyType.IsValidBucketPolicy() {
		return &json2.Error{
			Message: "Invalid policy type " + args.Policy,
		}
	}

	bucketPolicy, err := objectAPI.GetBucketPolicy(ctx, args.BucketName)
	if err != nil {
		if _, ok := err.(BucketPolicyNotFound); !ok {
			return toJSONError(ctx, err, args.BucketName)
		}
	}
	policyInfo, err := PolicyToBucketAccessPolicy(bucketPolicy)
	if err != nil {
		return toJSONError(ctx, err, args.BucketName)
	}

	policyInfo.Statements = iposgopolicy.SetPolicy(policyInfo.Statements, policyType, args.BucketName, args.Prefix)
	if len(policyInfo.Statements) == 0 {
		if err = objectAPI.DeleteBucketPolicy(ctx, args.BucketName); err != nil {
			return toJSONError(ctx, err, args.BucketName)
		}

		globalPolicySys.Remove(args.BucketName)
		return nil
	}

	bucketPolicy, err = BucketAccessPolicyToPolicy(policyInfo)
	if err != nil {
		return toJSONError(ctx, err, args.BucketName)
	}

	if err := objectAPI.SetBucketPolicy(ctx, args.BucketName, bucketPolicy); err != nil {
		return toJSONError(ctx, err, args.BucketName)
	}

	globalPolicySys.Set(args.BucketName, *bucketPolicy)

	return nil
}

type PresignedGetArgs struct {
	HostName string `json:"host"`

	BucketName string `json:"bucket"`

	ObjectName string `json:"object"`

	Expiry int64 `json:"expiry"`
}

type PresignedGetRep struct {
	UIVersion string `json:"uiVersion"`
	URL       string `json:"url"`
}

func (web *webAPIHandlers) PresignedGet(r *http.Request, args *PresignedGetArgs, reply *PresignedGetRep) error {
	ctx := newWebContext(r, args, "WebPresignedGet")
	claims, owner, authErr := webRequestAuthenticate(r)
	if authErr != nil {
		return toJSONError(ctx, authErr)
	}
	var creds auth.Credentials
	if !owner {
		var ok bool
		creds, ok = globalIAMSys.GetUser(claims.AccessKey)
		if !ok {
			return toJSONError(ctx, errInvalidAccessKeyID)
		}
	} else {
		creds = globalActiveCred
	}

	region := globalServerRegion
	if args.BucketName == "" || args.ObjectName == "" {
		return &json2.Error{
			Message: "Bucket and Object are mandatory arguments.",
		}
	}

	if isReservedOrInvalidBucket(args.BucketName, false) {
		return toJSONError(ctx, errInvalidBucketName)
	}

	reply.UIVersion = browser.UIVersion
	reply.URL = presignedGet(args.HostName, args.BucketName, args.ObjectName, args.Expiry, creds, region)
	return nil
}

func presignedGet(host, bucket, object string, expiry int64, creds auth.Credentials, region string) string {
	accessKey := creds.AccessKey
	secretKey := creds.SecretKey

	date := UTCNow()
	dateStr := date.Format(iso8601Format)
	credential := fmt.Sprintf("%s/%s", accessKey, getScope(date, region))

	var expiryStr = "604800"
	if expiry < 604800 && expiry > 0 {
		expiryStr = strconv.FormatInt(expiry, 10)
	}

	query := url.Values{}
	query.Set(xhttp.AmzAlgorithm, signV4Algorithm)
	query.Set(xhttp.AmzCredential, credential)
	query.Set(xhttp.AmzDate, dateStr)
	query.Set(xhttp.AmzExpires, expiryStr)
	query.Set(xhttp.AmzSignedHeaders, "host")
	queryStr := s3utils.QueryEncode(query)

	path := SlashSeparator + path.Join(bucket, object)

	extractedSignedHeaders := make(http.Header)
	extractedSignedHeaders.Set("host", host)
	canonicalRequest := getCanonicalRequest(extractedSignedHeaders, unsignedPayload, queryStr, path, http.MethodGet)
	stringToSign := getStringToSign(canonicalRequest, date, getScope(date, region))
	signingKey := getSigningKey(secretKey, date, region, serviceS3)
	signature := getSignature(signingKey, stringToSign)

	if creds.SessionToken != "" {
		return host + s3utils.EncodePath(path) + "?" + queryStr + "&" + xhttp.AmzSignature + "=" + signature + "&" + xhttp.AmzSecurityToken + "=" + creds.SessionToken
	}
	return host + s3utils.EncodePath(path) + "?" + queryStr + "&" + xhttp.AmzSignature + "=" + signature
}

type LoginSTSArgs struct {
	Token string `json:"token" form:"token"`
}

type LoginSTSRep struct {
	Token     string `json:"token"`
	UIVersion string `json:"uiVersion"`
}

func (web *webAPIHandlers) LoginSTS(r *http.Request, args *LoginSTSArgs, reply *LoginRep) error {
	ctx := newWebContext(r, args, "WebLoginSTS")

	v := url.Values{}
	v.Set("Action", webIdentity)
	v.Set("WebIdentityToken", args.Token)
	v.Set("Version", stsAPIVersion)

	scheme := "http"
	if globalIsSSL {
		scheme = "https"
	}

	u := &url.URL{
		Scheme: scheme,
		Host:   r.Host,
	}

	u.RawQuery = v.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return toJSONError(ctx, err)
	}

	clnt := &http.Client{
		Transport: NewGatewayHTTPTransport(),
	}
	defer clnt.CloseIdleConnections()

	resp, err := clnt.Do(req)
	if err != nil {
		return toJSONError(ctx, err)
	}
	defer xhttp.DrainBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return toJSONError(ctx, errors.New(resp.Status))
	}

	a := AssumeRoleWithWebIdentityResponse{}
	if err = xml.NewDecoder(resp.Body).Decode(&a); err != nil {
		return toJSONError(ctx, err)
	}

	reply.Token = a.Result.Credentials.SessionToken
	reply.UIVersion = browser.UIVersion
	return nil
}

func toJSONError(ctx context.Context, err error, params ...string) (jerr *json2.Error) {
	apiErr := toWebAPIError(ctx, err)
	jerr = &json2.Error{
		Message: apiErr.Description,
	}
	switch apiErr.Code {
	case "AllAccessDisabled":
		if len(params) > 0 {
			jerr = &json2.Error{
				Message: fmt.Sprintf("All access to this bucket %s has been disabled.", params[0]),
			}
		}
	case "InvalidBucketName":
		if len(params) > 0 {
			jerr = &json2.Error{
				Message: fmt.Sprintf("Bucket Name %s is invalid. Lowercase letters, period, hyphen, numerals are the only allowed characters and should be minimum 3 characters in length.", params[0]),
			}
		}
	case "NoSuchBucket":
		if len(params) > 0 {
			jerr = &json2.Error{
				Message: fmt.Sprintf("The specified bucket %s does not exist.", params[0]),
			}
		}
	case "NoSuchKey":
		if len(params) > 1 {
			jerr = &json2.Error{
				Message: fmt.Sprintf("The specified key %s does not exist", params[1]),
			}
		}
	}
	return jerr
}

func toWebAPIError(ctx context.Context, err error) APIError {
	switch err {
	case errNoAuthToken:
		return APIError{
			Code:           "WebTokenMissing",
			HTTPStatusCode: http.StatusBadRequest,
			Description:    err.Error(),
		}
	case errServerNotInitialized:
		return APIError{
			Code:           "XIPOSServerNotInitialized",
			HTTPStatusCode: http.StatusServiceUnavailable,
			Description:    err.Error(),
		}
	case errAuthentication, auth.ErrInvalidAccessKeyLength,
		auth.ErrInvalidSecretKeyLength, errInvalidAccessKeyID:
		return APIError{
			Code:           "AccessDenied",
			HTTPStatusCode: http.StatusForbidden,
			Description:    err.Error(),
		}
	case errSizeUnspecified:
		return APIError{
			Code:           "InvalidRequest",
			HTTPStatusCode: http.StatusBadRequest,
			Description:    err.Error(),
		}
	case errChangeCredNotAllowed:
		return APIError{
			Code:           "MethodNotAllowed",
			HTTPStatusCode: http.StatusMethodNotAllowed,
			Description:    err.Error(),
		}
	case errInvalidBucketName:
		return APIError{
			Code:           "InvalidBucketName",
			HTTPStatusCode: http.StatusBadRequest,
			Description:    err.Error(),
		}
	case errInvalidArgument:
		return APIError{
			Code:           "InvalidArgument",
			HTTPStatusCode: http.StatusBadRequest,
			Description:    err.Error(),
		}
	case errEncryptedObject:
		return getAPIError(ErrSSEEncryptedObject)
	case errInvalidEncryptionParameters:
		return getAPIError(ErrInvalidEncryptionParameters)
	case errObjectTampered:
		return getAPIError(ErrObjectTampered)
	case errMethodNotAllowed:
		return getAPIError(ErrMethodNotAllowed)
	}

	switch err.(type) {
	case StorageFull:
		return getAPIError(ErrStorageFull)
	case BucketNotFound:
		return getAPIError(ErrNoSuchBucket)
	case BucketNotEmpty:
		return getAPIError(ErrBucketNotEmpty)
	case BucketExists:
		return getAPIError(ErrBucketAlreadyOwnedByYou)
	case BucketNameInvalid:
		return getAPIError(ErrInvalidBucketName)
	case hash.BadDigest:
		return getAPIError(ErrBadDigest)
	case IncompleteBody:
		return getAPIError(ErrIncompleteBody)
	case ObjectExistsAsDirectory:
		return getAPIError(ErrObjectExistsAsDirectory)
	case ObjectNotFound:
		return getAPIError(ErrNoSuchKey)
	case ObjectNameInvalid:
		return getAPIError(ErrNoSuchKey)
	case InsufficientWriteQuorum:
		return getAPIError(ErrWriteQuorum)
	case InsufficientReadQuorum:
		return getAPIError(ErrReadQuorum)
	case NotImplemented:
		return APIError{
			Code:           "NotImplemented",
			HTTPStatusCode: http.StatusBadRequest,
			Description:    "Functionality not implemented",
		}
	}

	logger.LogIf(ctx, err)
	return toAPIError(ctx, err)
}

func writeWebErrorResponse(w http.ResponseWriter, err error) {
	reqInfo := &logger.ReqInfo{
		DeploymentID: globalDeploymentID,
	}
	ctx := logger.SetReqInfo(GlobalContext, reqInfo)
	apiErr := toWebAPIError(ctx, err)
	w.WriteHeader(apiErr.HTTPStatusCode)
	w.Write([]byte(apiErr.Description))
}
