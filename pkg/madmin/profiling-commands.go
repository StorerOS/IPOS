package madmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type ProfilerType string

const (
	ProfilerCPU        ProfilerType = "cpu"
	ProfilerMEM        ProfilerType = "mem"
	ProfilerBlock      ProfilerType = "block"
	ProfilerMutex      ProfilerType = "mutex"
	ProfilerTrace      ProfilerType = "trace"
	ProfilerThreads    ProfilerType = "threads"
	ProfilerGoroutines ProfilerType = "goroutines"
)

type StartProfilingResult struct {
	NodeName string `json:"nodeName"`
	Success  bool   `json:"success"`
	Error    string `json:"error"`
}

func (adm *AdminClient) StartProfiling(ctx context.Context, profiler ProfilerType) ([]StartProfilingResult, error) {
	v := url.Values{}
	v.Set("profilerType", string(profiler))
	resp, err := adm.executeMethod(ctx,
		http.MethodPost, requestData{
			relPath:     adminAPIPrefix + "/profiling/start",
			queryValues: v,
		},
	)
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	jsonResult, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var startResults []StartProfilingResult
	err = json.Unmarshal(jsonResult, &startResults)
	if err != nil {
		return nil, err
	}

	return startResults, nil
}

func (adm *AdminClient) DownloadProfilingData(ctx context.Context) (io.ReadCloser, error) {
	path := fmt.Sprintf(adminAPIPrefix + "/profiling/download")
	resp, err := adm.executeMethod(ctx,
		http.MethodGet, requestData{
			relPath: path,
		},
	)

	if err != nil {
		closeResponse(resp)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	if resp.Body == nil {
		return nil, errors.New("body is nil")
	}

	return resp.Body, nil
}
