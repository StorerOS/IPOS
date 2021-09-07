package cmd

import (
	"context"
	"io"
	"net/http"

	bucketsse "github.com/storeros/ipos/pkg/bucket/encryption"
	"github.com/storeros/ipos/pkg/bucket/lifecycle"
	"github.com/storeros/ipos/pkg/bucket/object/tagging"
	"github.com/storeros/ipos/pkg/bucket/policy"
	"github.com/storeros/ipos/pkg/encrypt"
	"github.com/storeros/ipos/pkg/madmin"
)

type CheckCopyPreconditionFn func(o ObjectInfo, encETag string) bool

type GetObjectInfoFn func(ctx context.Context, bucket, object string, opts ObjectOptions) (objInfo ObjectInfo, err error)

type ObjectOptions struct {
	ServerSideEncryption encrypt.ServerSide
	UserDefined          map[string]string
	PartNumber           int
	CheckCopyPrecondFn   CheckCopyPreconditionFn
}

type LockType int

const (
	noLock LockType = iota
	readLock
	writeLock
)

type ObjectLayer interface {
	Shutdown(context.Context) error
	CrawlAndGetDataUsage(ctx context.Context, updates chan<- DataUsageInfo) error
	StorageInfo(ctx context.Context, local bool) StorageInfo

	MakeBucketWithLocation(ctx context.Context, bucket string, location string) error
	GetBucketInfo(ctx context.Context, bucket string) (bucketInfo BucketInfo, err error)
	ListBuckets(ctx context.Context) (buckets []BucketInfo, err error)
	DeleteBucket(ctx context.Context, bucket string, forceDelete bool) error
	ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (result ListObjectsInfo, err error)
	ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (result ListObjectsV2Info, err error)
	Walk(ctx context.Context, bucket, prefix string, results chan<- ObjectInfo) error

	GetObjectNInfo(ctx context.Context, bucket, object string, rs *HTTPRangeSpec, h http.Header, lockType LockType, opts ObjectOptions) (reader *GetObjectReader, err error)
	GetObject(ctx context.Context, bucket, object string, startOffset int64, length int64, writer io.Writer, etag string, opts ObjectOptions) (err error)
	GetObjectInfo(ctx context.Context, bucket, object string, opts ObjectOptions) (objInfo ObjectInfo, err error)
	PutObject(ctx context.Context, bucket, object string, data *PutObjReader, opts ObjectOptions) (objInfo ObjectInfo, err error)
	CopyObject(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, srcInfo ObjectInfo, srcOpts, dstOpts ObjectOptions) (objInfo ObjectInfo, err error)
	DeleteObject(ctx context.Context, bucket, object string) error
	DeleteObjects(ctx context.Context, bucket string, objects []string) ([]error, error)

	ListMultipartUploads(ctx context.Context, bucket, prefix, keyMarker, uploadIDMarker, delimiter string, maxUploads int) (result ListMultipartsInfo, err error)
	NewMultipartUpload(ctx context.Context, bucket, object string, opts ObjectOptions) (uploadID string, err error)
	CopyObjectPart(ctx context.Context, srcBucket, srcObject, destBucket, destObject string, uploadID string, partID int,
		startOffset int64, length int64, srcInfo ObjectInfo, srcOpts, dstOpts ObjectOptions) (info PartInfo, err error)
	PutObjectPart(ctx context.Context, bucket, object, uploadID string, partID int, data *PutObjReader, opts ObjectOptions) (info PartInfo, err error)
	ListObjectParts(ctx context.Context, bucket, object, uploadID string, partNumberMarker int, maxParts int, opts ObjectOptions) (result ListPartsInfo, err error)
	AbortMultipartUpload(ctx context.Context, bucket, object, uploadID string) error
	CompleteMultipartUpload(ctx context.Context, bucket, object, uploadID string, uploadedParts []CompletePart, opts ObjectOptions) (objInfo ObjectInfo, err error)

	ReloadFormat(ctx context.Context, dryRun bool) error
	HealFormat(ctx context.Context, dryRun bool) (madmin.HealResultItem, error)
	HealBucket(ctx context.Context, bucket string, dryRun, remove bool) (madmin.HealResultItem, error)
	HealObject(ctx context.Context, bucket, object string, opts madmin.HealOpts) (madmin.HealResultItem, error)

	ListBucketsHeal(ctx context.Context) (buckets []BucketInfo, err error)

	SetBucketPolicy(context.Context, string, *policy.Policy) error
	GetBucketPolicy(context.Context, string) (*policy.Policy, error)
	DeleteBucketPolicy(context.Context, string) error

	IsNotificationSupported() bool
	IsListenBucketSupported() bool
	IsEncryptionSupported() bool

	IsCompressionSupported() bool

	SetBucketLifecycle(context.Context, string, *lifecycle.Lifecycle) error
	GetBucketLifecycle(context.Context, string) (*lifecycle.Lifecycle, error)
	DeleteBucketLifecycle(context.Context, string) error

	SetBucketSSEConfig(context.Context, string, *bucketsse.BucketSSEConfig) error
	GetBucketSSEConfig(context.Context, string) (*bucketsse.BucketSSEConfig, error)
	DeleteBucketSSEConfig(context.Context, string) error

	IsReady(ctx context.Context) bool

	PutObjectTag(context.Context, string, string, string) error
	GetObjectTag(context.Context, string, string) (tagging.Tagging, error)
	DeleteObjectTag(context.Context, string, string) error
}
