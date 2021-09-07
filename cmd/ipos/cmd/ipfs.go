package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	shell "github.com/ipfs/go-ipfs-api"

	"github.com/storeros/ipos/cmd/ipos/logger"
	bucketsse "github.com/storeros/ipos/pkg/bucket/encryption"
	"github.com/storeros/ipos/pkg/bucket/lifecycle"
	"github.com/storeros/ipos/pkg/bucket/object/tagging"
	"github.com/storeros/ipos/pkg/bucket/policy"
	"github.com/storeros/ipos/pkg/madmin"
	"github.com/storeros/ipos/pkg/s3utils"
)

func NewIPFSObjectLayer(host string) (ObjectLayer, error) {
	s := shell.NewShell(host)

	ipfs := IPFSObjects{
		shell:    s,
		listPool: NewTreeWalkPool(globalLookupTimeout),
	}

	if err := ipfs.initMetaVolumeFS(); err != nil {
		return nil, err
	}

	return &ipfs, nil
}

type IPFSObjects struct {
	shell *shell.Shell

	listPool *TreeWalkPool
}

func (fs *IPFSObjects) ipfsToObjectError(err error, params ...string) error {
	if err == nil {
		return nil
	}

	bucket := ""
	object := ""
	switch len(params) {
	case 2:
		object = params[1]
		fallthrough
	case 1:
		bucket = params[0]
	}

	switch {
	case strings.Contains(err.Error(), "file already exists"):
		if object != "" {
			return ObjectAlreadyExists{Bucket: bucket, Object: object}
		}
		return BucketAlreadyExists{Bucket: bucket}
	case strings.Contains(err.Error(), "file does not exist"):
		if object != "" {
			return ObjectNotFound{Bucket: bucket, Object: object}
		}
		return BucketNotFound{Bucket: bucket}
	default:
		logger.LogIf(context.Background(), err)
		return err
	}
}

func (fs *IPFSObjects) path(params ...string) string {
	bucket := ""
	object := ""
	switch len(params) {
	case 2:
		object = params[1]
		fallthrough
	case 1:
		bucket = params[0]
	}

	if bucket == "" {
		return "/"
	} else if object == "" {
		return fmt.Sprintf("/%s", bucket)
	} else {
		return fmt.Sprintf("/%s/%s", bucket, object)
	}
}

func (fs *IPFSObjects) initMetaVolumeFS() error {
	metaBucketPath := fs.path(iposMetaBucket)
	err := fs.shell.FilesMkdir(GlobalContext, metaBucketPath, shell.FilesMkdir.Parents(true))
	if err != nil {
		return fs.ipfsToObjectError(err, iposMetaBucket)
	}
	return nil
}

func (fs *IPFSObjects) NewNSLock(ctx context.Context, bucket string, objects ...string) RWLocker {
	return nil
}

func (fs *IPFSObjects) Shutdown(ctx context.Context) error {
	return nil
}

func (fs *IPFSObjects) StorageInfo(ctx context.Context, _ bool) StorageInfo {
	storageInfo := StorageInfo{}
	storageInfo.Backend.Type = BackendIPFS
	return storageInfo
}

func (fs *IPFSObjects) CrawlAndGetDataUsage(ctx context.Context, updates chan<- DataUsageInfo) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) MakeBucketWithLocation(ctx context.Context, bucket, location string) error {
	if s3utils.CheckValidBucketNameStrict(bucket) != nil {
		return BucketNameInvalid{Bucket: bucket}
	}
	path := fs.path(bucket)
	err := fs.shell.FilesMkdir(ctx, path)
	if err != nil {
		return fs.ipfsToObjectError(err, bucket)
	}

	return nil
}

func (fs *IPFSObjects) GetBucketInfo(ctx context.Context, bucket string) (bi BucketInfo, err error) {
	path := fs.path(bucket)
	_, err = fs.shell.FilesStat(ctx, path)
	if err != nil {
		return bi, fs.ipfsToObjectError(err, bucket)
	}

	return BucketInfo{
		Name:    bucket,
		Created: time.Now(),
	}, nil
}

func (fs *IPFSObjects) ListBuckets(ctx context.Context) (buckets []BucketInfo, err error) {
	path := fs.path()
	list, err := fs.shell.FilesLs(ctx, path, shell.FilesLs.Stat(true))
	if err != nil {
		return buckets, fs.ipfsToObjectError(err)
	}

	for _, entry := range list {
		if isReservedOrInvalidBucket(entry.Name, false) {
			continue
		}
		buckets = append(buckets, BucketInfo{
			Name:    entry.Name,
			Created: time.Now(),
		})
	}

	return buckets, nil
}

func (fs *IPFSObjects) DeleteBucket(ctx context.Context, bucket string, forceDelete bool) error {
	path := fs.path(bucket)
	_, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return fs.ipfsToObjectError(err, bucket)
	}

	err = fs.shell.FilesRm(ctx, path, true)
	if err != nil {
		return fs.ipfsToObjectError(err, bucket)
	}

	return nil
}

func (fs *IPFSObjects) CopyObject(ctx context.Context, srcBucket, srcObject, dstBucket, dstObject string, srcInfo ObjectInfo, srcOpts, dstOpts ObjectOptions) (oi ObjectInfo, e error) {
	logger.LogIf(ctx, NotImplemented{})
	return ObjectInfo{}, NotImplemented{}
}

func (fs *IPFSObjects) GetObjectNInfo(ctx context.Context, bucket, object string, rs *HTTPRangeSpec, h http.Header, lockType LockType, opts ObjectOptions) (gr *GetObjectReader, err error) {
	objInfo, err := fs.GetObjectInfo(ctx, bucket, object, opts)
	if err != nil {
		return nil, err
	}

	var startOffset, length int64
	startOffset, length, err = rs.GetOffsetLength(objInfo.Size)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		nerr := fs.GetObject(ctx, bucket, object, startOffset, length, pw, objInfo.ETag, opts)
		pw.CloseWithError(nerr)
	}()

	pipeCloser := func() { pr.Close() }
	return NewGetObjectReaderFromReader(pr, objInfo, opts, pipeCloser)
}

func (fs *IPFSObjects) GetObject(ctx context.Context, bucket, object string, offset int64, length int64, writer io.Writer, etag string, opts ObjectOptions) error {
	path := fs.path(bucket)
	_, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return fs.ipfsToObjectError(err, bucket)
	}

	path = fs.path(bucket, object)
	reader, err := fs.shell.FilesRead(ctx, path)
	if err != nil {
		return fs.ipfsToObjectError(err, bucket, object)
	}
	_, err = io.Copy(writer, reader)
	if err != nil {
		return fs.ipfsToObjectError(err, bucket, object)
	}

	return nil
}

func (fs *IPFSObjects) GetObjectInfo(ctx context.Context, bucket, object string, opts ObjectOptions) (objInfo ObjectInfo, e error) {
	path := fs.path(bucket, object)
	stat, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return objInfo, fs.ipfsToObjectError(err, bucket, object)
	}
	return ObjectInfo{
		Bucket:  bucket,
		Name:    object,
		ETag:    stat.Hash,
		ModTime: time.Now(),
		Size:    int64(stat.Size),
		IsDir:   (stat.Type == "directory"),
		AccTime: time.Now(),
	}, nil
}

func (fs *IPFSObjects) PutObject(ctx context.Context, bucket string, object string, r *PutObjReader, opts ObjectOptions) (objInfo ObjectInfo, retErr error) {
	path := fs.path(bucket)
	_, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return objInfo, fs.ipfsToObjectError(err, bucket)
	}

	path = fs.path(bucket, object)
	err = fs.shell.FilesWrite(ctx, path, r, shell.FilesWrite.Parents(true), shell.FilesWrite.Create(true))
	if err != nil {
		return objInfo, fs.ipfsToObjectError(err, bucket, object)
	}

	stat, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return objInfo, fs.ipfsToObjectError(err, bucket, object)
	}

	return ObjectInfo{
		Bucket:  bucket,
		Name:    object,
		ETag:    stat.Hash,
		ModTime: time.Now(),
		Size:    int64(stat.Size),
		IsDir:   (stat.Type == "directory"),
		AccTime: time.Now(),
	}, nil
}

func (fs *IPFSObjects) DeleteObjects(ctx context.Context, bucket string, objects []string) ([]error, error) {
	errs := make([]error, len(objects))
	for idx, object := range objects {
		errs[idx] = fs.DeleteObject(ctx, bucket, object)
	}
	return errs, nil
}

func (fs *IPFSObjects) deleteObject(ctx context.Context, bucket, object string) error {
	if object == "" || object == "." || object == SlashSeparator {
		return nil
	}

	path := fs.path(bucket, object)
	stat, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return err
	}

	if stat.Type == "directory" && stat.Blocks != 0 {
		return nil
	}

	return fs.shell.FilesRm(ctx, path, true)
}

func (fs *IPFSObjects) DeleteObject(ctx context.Context, bucket, object string) error {
	path := fs.path(bucket)
	_, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return fs.ipfsToObjectError(err, bucket)
	}

	path = fs.path(bucket, object)
	err = fs.deleteObject(ctx, bucket, object)
	if err != nil {
		return fs.ipfsToObjectError(err, bucket, object)
	}

	return nil
}

func (fs *IPFSObjects) listDirFactory() ListDirFunc {
	listDir := func(bucket, prefixDir, prefixEntry string) (emptyDir bool, entries []string) {
		path := fs.path(bucket, prefixDir)
		list, err := fs.shell.FilesLs(GlobalContext, path, shell.FilesLs.Stat(true))
		if err != nil {
			if strings.Contains(err.Error(), "file does not exist") {
				return false, nil
			}
			logger.LogIf(GlobalContext, err)
			return
		}
		if len(list) == 0 {
			return true, nil
		}
		for _, entry := range list {
			object := prefixDir + "/" + entry.Name
			path := fs.path(bucket, object)
			stat, err := fs.shell.FilesStat(GlobalContext, path)
			if err == nil {
				if stat.Type == "directory" {
					entries = append(entries, entry.Name+SlashSeparator)
				} else {
					entries = append(entries, entry.Name)
				}
			}

		}
		return false, filterMatchingPrefix(entries, prefixEntry)
	}

	return listDir
}

func (fs *IPFSObjects) getObjectInfo(ctx context.Context, bucket, object string) (objInfo ObjectInfo, err error) {
	path := fs.path(bucket, object)
	stat, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return objInfo, fs.ipfsToObjectError(err, bucket, object)
	}

	return ObjectInfo{
		Bucket:  bucket,
		Name:    object,
		ETag:    stat.Hash,
		ModTime: time.Now(),
		Size:    int64(stat.Size),
		IsDir:   (stat.Type == "directory"),
		AccTime: time.Now(),
	}, nil
}

func (fs *IPFSObjects) ListObjects(ctx context.Context, bucket, prefix, marker, delimiter string, maxKeys int) (loi ListObjectsInfo, e error) {
	path := fs.path(bucket)
	_, err := fs.shell.FilesStat(ctx, path)
	if err != nil {
		return loi, fs.ipfsToObjectError(err, bucket)
	}

	return listObjects(ctx, fs, bucket, prefix, marker, delimiter, maxKeys, fs.listPool, fs.listDirFactory(), fs.getObjectInfo, fs.getObjectInfo)
}

func (fs *IPFSObjects) GetObjectTag(ctx context.Context, bucket, object string) (tagging.Tagging, error) {
	logger.LogIf(ctx, NotImplemented{})
	return tagging.Tagging{}, NotImplemented{}
}

func (fs *IPFSObjects) PutObjectTag(ctx context.Context, bucket, object string, tags string) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) DeleteObjectTag(ctx context.Context, bucket, object string) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) ReloadFormat(ctx context.Context, dryRun bool) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) HealFormat(ctx context.Context, dryRun bool) (madmin.HealResultItem, error) {
	logger.LogIf(ctx, NotImplemented{})
	return madmin.HealResultItem{}, NotImplemented{}
}

func (fs *IPFSObjects) HealObject(ctx context.Context, bucket, object string, opts madmin.HealOpts) (
	res madmin.HealResultItem, err error) {
	logger.LogIf(ctx, NotImplemented{})
	return res, NotImplemented{}
}

func (fs *IPFSObjects) HealBucket(ctx context.Context, bucket string, dryRun, remove bool) (madmin.HealResultItem, error) {
	logger.LogIf(ctx, NotImplemented{})
	return madmin.HealResultItem{}, NotImplemented{}
}

func (fs *IPFSObjects) Walk(ctx context.Context, bucket, prefix string, results chan<- ObjectInfo) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) ListBucketsHeal(ctx context.Context) ([]BucketInfo, error) {
	logger.LogIf(ctx, NotImplemented{})
	return []BucketInfo{}, NotImplemented{}
}

func (fs *IPFSObjects) GetMetrics(ctx context.Context) (*Metrics, error) {
	logger.LogIf(ctx, NotImplemented{})
	return nil, NotImplemented{}
}

func (fs *IPFSObjects) SetBucketPolicy(ctx context.Context, bucket string, policy *policy.Policy) error {
	return savePolicyConfig(ctx, fs, bucket, policy)
}

func (fs *IPFSObjects) GetBucketPolicy(ctx context.Context, bucket string) (*policy.Policy, error) {
	return getPolicyConfig(fs, bucket)
}

func (fs *IPFSObjects) DeleteBucketPolicy(ctx context.Context, bucket string) error {
	return removePolicyConfig(ctx, fs, bucket)
}

func (fs *IPFSObjects) SetBucketLifecycle(ctx context.Context, bucket string, lifecycle *lifecycle.Lifecycle) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) GetBucketLifecycle(ctx context.Context, bucket string) (*lifecycle.Lifecycle, error) {
	logger.LogIf(ctx, NotImplemented{})
	return nil, NotImplemented{}
}

func (fs *IPFSObjects) DeleteBucketLifecycle(ctx context.Context, bucket string) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) GetBucketSSEConfig(ctx context.Context, bucket string) (*bucketsse.BucketSSEConfig, error) {
	logger.LogIf(ctx, NotImplemented{})
	return nil, NotImplemented{}
}

func (fs *IPFSObjects) SetBucketSSEConfig(ctx context.Context, bucket string, config *bucketsse.BucketSSEConfig) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) DeleteBucketSSEConfig(ctx context.Context, bucket string) error {
	logger.LogIf(ctx, NotImplemented{})
	return NotImplemented{}
}

func (fs *IPFSObjects) ListObjectsV2(ctx context.Context, bucket, prefix, continuationToken, delimiter string, maxKeys int, fetchOwner bool, startAfter string) (loi ListObjectsV2Info, err error) {
	marker := continuationToken
	if marker == "" {
		marker = startAfter
	}
	resultV1, err := fs.ListObjects(ctx, bucket, prefix, marker, delimiter, maxKeys)
	if err != nil {
		return loi, err
	}
	return ListObjectsV2Info{
		Objects:               resultV1.Objects,
		Prefixes:              resultV1.Prefixes,
		ContinuationToken:     continuationToken,
		NextContinuationToken: resultV1.NextMarker,
		IsTruncated:           resultV1.IsTruncated,
	}, nil
}

func (fs *IPFSObjects) IsNotificationSupported() bool {
	return false
}

func (fs *IPFSObjects) IsListenBucketSupported() bool {
	return false
}

func (fs *IPFSObjects) IsEncryptionSupported() bool {
	return false
}

func (fs *IPFSObjects) IsCompressionSupported() bool {
	return true
}

func (fs *IPFSObjects) IsReady(_ context.Context) bool {
	return true
}
