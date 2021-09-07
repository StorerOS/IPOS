package madmin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type HealScanMode int

const (
	HealNormalScan HealScanMode = iota
	HealDeepScan
)

type HealOpts struct {
	Recursive bool         `json:"recursive"`
	DryRun    bool         `json:"dryRun"`
	Remove    bool         `json:"remove"`
	ScanMode  HealScanMode `json:"scanMode"`
}

type HealStartSuccess struct {
	ClientToken   string    `json:"clientToken"`
	ClientAddress string    `json:"clientAddress"`
	StartTime     time.Time `json:"startTime"`
}

type HealStopSuccess HealStartSuccess

type HealTaskStatus struct {
	Summary       string    `json:"summary"`
	FailureDetail string    `json:"detail"`
	StartTime     time.Time `json:"startTime"`
	HealSettings  HealOpts  `json:"settings"`
	NumDisks      int       `json:"numDisks"`

	Items []HealResultItem `json:"items,omitempty"`
}

type HealItemType string

const (
	HealItemMetadata       HealItemType = "metadata"
	HealItemBucket                      = "bucket"
	HealItemBucketMetadata              = "bucket-metadata"
	HealItemObject                      = "object"
)

const (
	DriveStateOk          string = "ok"
	DriveStateOffline            = "offline"
	DriveStateCorrupt            = "corrupt"
	DriveStateMissing            = "missing"
	DriveStateUnformatted        = "unformatted"
)

type HealDriveInfo struct {
	UUID     string `json:"uuid"`
	Endpoint string `json:"endpoint"`
	State    string `json:"state"`
}

type HealResultItem struct {
	ResultIndex  int64        `json:"resultId"`
	Type         HealItemType `json:"type"`
	Bucket       string       `json:"bucket"`
	Object       string       `json:"object"`
	Detail       string       `json:"detail"`
	ParityBlocks int          `json:"parityBlocks,omitempty"`
	DataBlocks   int          `json:"dataBlocks,omitempty"`
	DiskCount    int          `json:"diskCount"`
	SetCount     int          `json:"setCount"`
	Before       struct {
		Drives []HealDriveInfo `json:"drives"`
	} `json:"before"`
	After struct {
		Drives []HealDriveInfo `json:"drives"`
	} `json:"after"`
	ObjectSize int64 `json:"objectSize"`
}

func (hri *HealResultItem) GetMissingCounts() (b, a int) {
	if hri == nil {
		return
	}
	for _, v := range hri.Before.Drives {
		if v.State == DriveStateMissing {
			b++
		}
	}
	for _, v := range hri.After.Drives {
		if v.State == DriveStateMissing {
			a++
		}
	}
	return
}

func (hri *HealResultItem) GetOfflineCounts() (b, a int) {
	if hri == nil {
		return
	}
	for _, v := range hri.Before.Drives {
		if v.State == DriveStateOffline {
			b++
		}
	}
	for _, v := range hri.After.Drives {
		if v.State == DriveStateOffline {
			a++
		}
	}
	return
}

func (hri *HealResultItem) GetCorruptedCounts() (b, a int) {
	if hri == nil {
		return
	}
	for _, v := range hri.Before.Drives {
		if v.State == DriveStateCorrupt {
			b++
		}
	}
	for _, v := range hri.After.Drives {
		if v.State == DriveStateCorrupt {
			a++
		}
	}
	return
}

func (hri *HealResultItem) GetOnlineCounts() (b, a int) {
	if hri == nil {
		return
	}
	for _, v := range hri.Before.Drives {
		if v.State == DriveStateOk {
			b++
		}
	}
	for _, v := range hri.After.Drives {
		if v.State == DriveStateOk {
			a++
		}
	}
	return
}

func (adm *AdminClient) Heal(ctx context.Context, bucket, prefix string,
	healOpts HealOpts, clientToken string, forceStart, forceStop bool) (
	healStart HealStartSuccess, healTaskStatus HealTaskStatus, err error) {

	if forceStart && forceStop {
		return healStart, healTaskStatus, ErrInvalidArgument("forceStart and forceStop set to true is not allowed")
	}

	body, err := json.Marshal(healOpts)
	if err != nil {
		return healStart, healTaskStatus, err
	}

	path := fmt.Sprintf(adminAPIPrefix+"/heal/%s", bucket)
	if bucket != "" && prefix != "" {
		path += "/" + prefix
	}

	queryVals := make(url.Values)
	if clientToken != "" {
		queryVals.Set("clientToken", clientToken)
		body = []byte{}
	}

	if forceStart {
		queryVals.Set("forceStart", "true")
	} else if forceStop {
		queryVals.Set("forceStop", "true")
	}

	resp, err := adm.executeMethod(ctx,
		http.MethodPost, requestData{
			relPath:     path,
			content:     body,
			queryValues: queryVals,
		})
	defer closeResponse(resp)
	if err != nil {
		return healStart, healTaskStatus, err
	}

	if resp.StatusCode != http.StatusOK {
		return healStart, healTaskStatus, httpRespToErrorResponse(resp)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return healStart, healTaskStatus, err
	}

	if clientToken == "" {
		err = json.Unmarshal(respBytes, &healStart)
	} else {
		err = json.Unmarshal(respBytes, &healTaskStatus)
	}
	if err != nil {
		var errResp ErrorResponse
		err = json.Unmarshal(respBytes, &errResp)
		if err != nil {
			return healStart, healTaskStatus, err
		}
		return healStart, healTaskStatus, errResp
	}
	return healStart, healTaskStatus, nil
}

type BgHealState struct {
	ScannedItemsCount int64
	LastHealActivity  time.Time
	NextHealRound     time.Time
}

func (adm *AdminClient) BackgroundHealStatus(ctx context.Context) (BgHealState, error) {
	resp, err := adm.executeMethod(ctx,
		http.MethodPost,
		requestData{relPath: adminAPIPrefix + "/background-heal/status"})
	if err != nil {
		return BgHealState{}, err
	}
	defer closeResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return BgHealState{}, httpRespToErrorResponse(resp)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return BgHealState{}, err
	}

	var healState BgHealState

	err = json.Unmarshal(respBytes, &healState)
	if err != nil {
		return BgHealState{}, err
	}
	return healState, nil
}
