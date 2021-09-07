package madmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

type LogInfo struct {
	logEntry
	ConsoleMsg string
	NodeName   string `json:"node"`
	Err        error  `json:"-"`
}

func (adm AdminClient) GetLogs(ctx context.Context, node string, lineCnt int, logKind string) <-chan LogInfo {
	logCh := make(chan LogInfo, 1)

	go func(logCh chan<- LogInfo) {
		defer close(logCh)
		urlValues := make(url.Values)
		urlValues.Set("node", node)
		urlValues.Set("limit", strconv.Itoa(lineCnt))
		urlValues.Set("logType", logKind)
		for {
			reqData := requestData{
				relPath:     adminAPIPrefix + "/log",
				queryValues: urlValues,
			}
			resp, err := adm.executeMethod(ctx, http.MethodGet, reqData)
			if err != nil {
				closeResponse(resp)
				return
			}

			if resp.StatusCode != http.StatusOK {
				logCh <- LogInfo{Err: httpRespToErrorResponse(resp)}
				return
			}
			dec := json.NewDecoder(resp.Body)
			for {
				var info LogInfo
				if err = dec.Decode(&info); err != nil {
					break
				}
				select {
				case <-ctx.Done():
					return
				case logCh <- info:
				}
			}

		}
	}(logCh)

	return logCh
}
