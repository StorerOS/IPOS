package madmin

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math"
	"net/http"
	"time"
)

type BackendType int

const (
	Unknown BackendType = iota
	FS
	Erasure
)

type DriveInfo HealDriveInfo

type StorageInfo struct {
	Used []uint64

	Total []uint64

	Available []uint64

	MountPaths []string

	Backend struct {
		Type BackendType

		OnlineDisks      BackendDisks
		OfflineDisks     BackendDisks
		StandardSCData   int
		StandardSCParity int
		RRSCData         int
		RRSCParity       int

		Sets [][]DriveInfo
	}
}

type BackendDisks map[string]int

func (d1 BackendDisks) Sum() (sum int) {
	for _, count := range d1 {
		sum += count
	}
	return sum
}

func (d1 BackendDisks) Merge(d2 BackendDisks) BackendDisks {
	if len(d2) == 0 {
		d2 = make(BackendDisks)
	}
	for i1, v1 := range d1 {
		if v2, ok := d2[i1]; ok {
			d2[i1] = v2 + v1
			continue
		}
		d2[i1] = v1
	}
	return d2
}

func (adm *AdminClient) StorageInfo(ctx context.Context) (StorageInfo, error) {
	resp, err := adm.executeMethod(ctx, http.MethodGet, requestData{relPath: adminAPIPrefix + "/storageinfo"})
	defer closeResponse(resp)
	if err != nil {
		return StorageInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return StorageInfo{}, httpRespToErrorResponse(resp)
	}

	var storageInfo StorageInfo

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return StorageInfo{}, err
	}

	err = json.Unmarshal(respBytes, &storageInfo)
	if err != nil {
		return StorageInfo{}, err
	}

	return storageInfo, nil
}

type objectHistogramInterval struct {
	name       string
	start, end int64
}

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
	LastUpdate       time.Time `json:"lastUpdate"`
	ObjectsCount     uint64    `json:"objectsCount"`
	ObjectsTotalSize uint64    `json:"objectsTotalSize"`

	ObjectsSizesHistogram map[string]uint64 `json:"objectsSizesHistogram"`

	BucketsCount uint64 `json:"bucketsCount"`

	BucketsSizes map[string]uint64 `json:"bucketsSizes"`
}

func (adm *AdminClient) DataUsageInfo(ctx context.Context) (DataUsageInfo, error) {
	resp, err := adm.executeMethod(ctx, http.MethodGet, requestData{relPath: adminAPIPrefix + "/datausageinfo"})
	defer closeResponse(resp)
	if err != nil {
		return DataUsageInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return DataUsageInfo{}, httpRespToErrorResponse(resp)
	}

	var dataUsageInfo DataUsageInfo

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return DataUsageInfo{}, err
	}

	err = json.Unmarshal(respBytes, &dataUsageInfo)
	if err != nil {
		return DataUsageInfo{}, err
	}

	return dataUsageInfo, nil
}

type AccountAccess struct {
	AccountName string `json:"accountName"`
	Read        bool   `json:"read"`
	Write       bool   `json:"write"`
	Custom      bool   `json:"custom"`
}

type BucketAccountingUsage struct {
	Size       uint64          `json:"size"`
	AccessList []AccountAccess `json:"accessList"`
}

func (adm *AdminClient) AccountingUsageInfo(ctx context.Context) (map[string]BucketAccountingUsage, error) {
	resp, err := adm.executeMethod(ctx, http.MethodGet, requestData{relPath: adminAPIPrefix + "/accountingusageinfo"})
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	var accountingUsageInfo map[string]BucketAccountingUsage

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(respBytes, &accountingUsageInfo)
	if err != nil {
		return nil, err
	}

	return accountingUsageInfo, nil
}

type InfoMessage struct {
	Mode         string             `json:"mode,omitempty"`
	Domain       []string           `json:"domain,omitempty"`
	Region       string             `json:"region,omitempty"`
	SQSARN       []string           `json:"sqsARN,omitempty"`
	DeploymentID string             `json:"deploymentID,omitempty"`
	Buckets      Buckets            `json:"buckets,omitempty"`
	Objects      Objects            `json:"objects,omitempty"`
	Usage        Usage              `json:"usage,omitempty"`
	Services     Services           `json:"services,omitempty"`
	Backend      interface{}        `json:"backend,omitempty"`
	Servers      []ServerProperties `json:"servers,omitempty"`
}

type Services struct {
	Vault         Vault                         `json:"vault,omitempty"`
	LDAP          LDAP                          `json:"ldap,omitempty"`
	Logger        []Logger                      `json:"logger,omitempty"`
	Audit         []Audit                       `json:"audit,omitempty"`
	Notifications []map[string][]TargetIDStatus `json:"notifications,omitempty"`
}

type Buckets struct {
	Count uint64 `json:"count,omitempty"`
}

type Objects struct {
	Count uint64 `json:"count,omitempty"`
}

type Usage struct {
	Size uint64 `json:"size,omitempty"`
}

type Vault struct {
	Status  string `json:"status,omitempty"`
	Encrypt string `json:"encryp,omitempty"`
	Decrypt string `json:"decrypt,omitempty"`
}

type LDAP struct {
	Status string `json:"status,omitempty"`
}

type Status struct {
	Status string `json:"status,omitempty"`
}

type Audit map[string]Status

type Logger map[string]Status

type TargetIDStatus map[string]Status

type backendType string

const (
	FsType      = backendType("FS")
	ErasureType = backendType("Erasure")
)

type FSBackend struct {
	Type backendType `json:"backendType,omitempty"`
}

type XLBackend struct {
	Type             backendType `json:"backendType,omitempty"`
	OnlineDisks      int         `json:"onlineDisks,omitempty"`
	OfflineDisks     int         `json:"offlineDisks,omitempty"`
	StandardSCData   int         `json:"standardSCData,omitempty"`
	StandardSCParity int         `json:"standardSCParity,omitempty"`
	RRSCData         int         `json:"rrSCData,omitempty"`
	RRSCParity       int         `json:"rrSCParity,omitempty"`
}

type ServerProperties struct {
	State    string            `json:"state,omitempty"`
	Endpoint string            `json:"endpoint,omitempty"`
	Uptime   int64             `json:"uptime,omitempty"`
	Version  string            `json:"version,omitempty"`
	CommitID string            `json:"commitID,omitempty"`
	Network  map[string]string `json:"network,omitempty"`
	Disks    []Disk            `json:"disks,omitempty"`
}

type Disk struct {
	DrivePath       string  `json:"path,omitempty"`
	State           string  `json:"state,omitempty"`
	UUID            string  `json:"uuid,omitempty"`
	Model           string  `json:"model,omitempty"`
	TotalSpace      uint64  `json:"totalspace,omitempty"`
	UsedSpace       uint64  `json:"usedspace,omitempty"`
	ReadThroughput  float64 `json:"readthroughput,omitempty"`
	WriteThroughPut float64 `json:"writethroughput,omitempty"`
	ReadLatency     float64 `json:"readlatency,omitempty"`
	WriteLatency    float64 `json:"writelatency,omitempty"`
	Utilization     float64 `json:"utilization,omitempty"`
}

func (adm *AdminClient) ServerInfo(ctx context.Context) (InfoMessage, error) {
	resp, err := adm.executeMethod(ctx,
		http.MethodGet,
		requestData{relPath: adminAPIPrefix + "/info"},
	)
	defer closeResponse(resp)
	if err != nil {
		return InfoMessage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return InfoMessage{}, httpRespToErrorResponse(resp)
	}

	var message InfoMessage

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return InfoMessage{}, err
	}

	err = json.Unmarshal(respBytes, &message)
	if err != nil {
		return InfoMessage{}, err
	}

	return message, nil
}
