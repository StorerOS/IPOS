package cmd

import (
	"io"
	"math"
	"time"

	"github.com/storeros/ipos/pkg/hash"
	"github.com/storeros/ipos/pkg/madmin"
)

type BackendType int

const (
	Unknown BackendType = iota
	BackendIPFS
)

type StorageInfo struct {
	Used []uint64

	Total []uint64

	Available []uint64

	MountPaths []string

	Backend struct {
		Type BackendType

		GatewayOnline bool

		OnlineDisks      madmin.BackendDisks
		OfflineDisks     madmin.BackendDisks
		StandardSCData   int
		StandardSCParity int
		RRSCData         int
		RRSCParity       int

		Sets [][]madmin.DriveInfo
	}
}

type objectHistogramInterval struct {
	name       string
	start, end int64
}

const (
	dataUsageBucketLen = 7
)

var ObjectsHistogramIntervals = []objectHistogramInterval{
	{"LESS_THAN_1024_B", -1, 1024 - 1},
	{"BETWEEN_1024_B_AND_1_MB", 1024, 1024*1024 - 1},
	{"BETWEEN_1_MB_AND_10_MB", 1024 * 1024, 1024*1024*10 - 1},
	{"BETWEEN_10_MB_AND_64_MB", 1024 * 1024 * 10, 1024*1024*64 - 1},
	{"BETWEEN_64_MB_AND_128_MB", 1024 * 1024 * 64, 1024*1024*128 - 1},
	{"BETWEEN_128_MB_AND_512_MB", 1024 * 1024 * 128, 1024*1024*512 - 1},
	{"GREATER_THAN_512_MB", 1024 * 1024 * 512, math.MaxInt64},
}

type DataUsageInfo struct {
	LastUpdate time.Time `json:"lastUpdate"`

	ObjectsCount     uint64 `json:"objectsCount"`
	ObjectsTotalSize uint64 `json:"objectsTotalSize"`

	ObjectsSizesHistogram map[string]uint64 `json:"objectsSizesHistogram"`

	BucketsCount uint64            `json:"bucketsCount"`
	BucketsSizes map[string]uint64 `json:"bucketsSizes"`
}

type BucketInfo struct {
	Name string

	Created time.Time
}

type ObjectInfo struct {
	Bucket string

	Name string

	ModTime time.Time

	Size int64

	IsDir bool

	ETag string

	ContentType string

	ContentEncoding string

	Expires time.Time

	StorageClass string

	UserDefined map[string]string

	UserTags string

	Writer       io.WriteCloser `json:"-"`
	Reader       *hash.Reader   `json:"-"`
	PutObjReader *PutObjReader  `json:"-"`

	metadataOnly bool

	AccTime time.Time

	backendType BackendType
}

type ListPartsInfo struct {
	Bucket string

	Object string

	UploadID string

	StorageClass string

	PartNumberMarker int

	NextPartNumberMarker int

	MaxParts int

	IsTruncated bool

	Parts []PartInfo

	UserDefined map[string]string

	EncodingType string
}

func (lm ListMultipartsInfo) Lookup(uploadID string) bool {
	for _, upload := range lm.Uploads {
		if upload.UploadID == uploadID {
			return true
		}
	}
	return false
}

type ListMultipartsInfo struct {
	KeyMarker string

	UploadIDMarker string

	NextKeyMarker string

	NextUploadIDMarker string

	MaxUploads int

	IsTruncated bool

	Uploads []MultipartInfo

	Prefix string

	Delimiter string

	CommonPrefixes []string

	EncodingType string
}

type ListObjectsInfo struct {
	IsTruncated bool

	NextMarker string

	Objects []ObjectInfo

	Prefixes []string
}

type ListObjectsV2Info struct {
	IsTruncated bool

	ContinuationToken     string
	NextContinuationToken string

	Objects []ObjectInfo

	Prefixes []string
}

type PartInfo struct {
	PartNumber int

	LastModified time.Time

	ETag string

	Size int64

	ActualSize int64
}

type MultipartInfo struct {
	Object string

	UploadID string

	Initiated time.Time

	StorageClass string
}

type CompletePart struct {
	PartNumber int

	ETag string
}

type CompletedParts []CompletePart

func (a CompletedParts) Len() int           { return len(a) }
func (a CompletedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a CompletedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

type CompleteMultipartUpload struct {
	Parts []CompletePart `xml:"Part"`
}
