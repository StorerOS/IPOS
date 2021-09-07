package madmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	trace "github.com/storeros/ipos/pkg/trace"
)

func (adm *AdminClient) ServiceRestart(ctx context.Context) error {
	return adm.serviceCallAction(ctx, ServiceActionRestart)
}

func (adm *AdminClient) ServiceStop(ctx context.Context) error {
	return adm.serviceCallAction(ctx, ServiceActionStop)
}

type ServiceAction string

const (
	ServiceActionRestart ServiceAction = "restart"
	ServiceActionStop                  = "stop"
)

func (adm *AdminClient) serviceCallAction(ctx context.Context, action ServiceAction) error {
	queryValues := url.Values{}
	queryValues.Set("action", string(action))

	resp, err := adm.executeMethod(ctx,
		http.MethodPost, requestData{
			relPath:     adminAPIPrefix + "/service",
			queryValues: queryValues,
		},
	)
	defer closeResponse(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return httpRespToErrorResponse(resp)
	}

	return nil
}

type ServiceTraceInfo struct {
	Trace trace.Info
	Err   error `json:"-"`
}

func (adm AdminClient) ServiceTrace(ctx context.Context, allTrace, errTrace bool) <-chan ServiceTraceInfo {
	traceInfoCh := make(chan ServiceTraceInfo)
	go func(traceInfoCh chan<- ServiceTraceInfo) {
		defer close(traceInfoCh)
		for {
			urlValues := make(url.Values)
			urlValues.Set("all", strconv.FormatBool(allTrace))
			urlValues.Set("err", strconv.FormatBool(errTrace))
			reqData := requestData{
				relPath:     adminAPIPrefix + "/trace",
				queryValues: urlValues,
			}
			resp, err := adm.executeMethod(ctx, http.MethodGet, reqData)
			if err != nil {
				closeResponse(resp)
				return
			}

			if resp.StatusCode != http.StatusOK {
				traceInfoCh <- ServiceTraceInfo{Err: httpRespToErrorResponse(resp)}
				return
			}

			dec := json.NewDecoder(resp.Body)
			for {
				var info trace.Info
				if err = dec.Decode(&info); err != nil {
					break
				}
				select {
				case <-ctx.Done():
					return
				case traceInfoCh <- ServiceTraceInfo{Trace: info}:
				}
			}
		}
	}(traceInfoCh)

	return traceInfoCh
}
