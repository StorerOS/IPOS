package madmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (adm *AdminClient) ClearConfigHistoryKV(ctx context.Context, restoreID string) (err error) {
	v := url.Values{}
	v.Set("restoreId", restoreID)
	reqData := requestData{
		relPath:     adminAPIPrefix + "/clear-config-history-kv",
		queryValues: v,
	}

	resp, err := adm.executeMethod(ctx, http.MethodDelete, reqData)

	defer closeResponse(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return httpRespToErrorResponse(resp)
	}

	return nil
}

func (adm *AdminClient) RestoreConfigHistoryKV(ctx context.Context, restoreID string) (err error) {
	v := url.Values{}
	v.Set("restoreId", restoreID)
	reqData := requestData{
		relPath:     adminAPIPrefix + "/restore-config-history-kv",
		queryValues: v,
	}

	resp, err := adm.executeMethod(ctx, http.MethodPut, reqData)

	defer closeResponse(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return httpRespToErrorResponse(resp)
	}

	return nil
}

type ConfigHistoryEntry struct {
	RestoreID  string    `json:"restoreId"`
	CreateTime time.Time `json:"createTime"`
	Data       string    `json:"data"`
}

func (ch ConfigHistoryEntry) CreateTimeFormatted() string {
	return ch.CreateTime.Format(http.TimeFormat)
}

func (adm *AdminClient) ListConfigHistoryKV(ctx context.Context, count int) ([]ConfigHistoryEntry, error) {
	if count == 0 {
		count = 10
	}
	v := url.Values{}
	v.Set("count", strconv.Itoa(count))

	resp, err := adm.executeMethod(ctx,
		http.MethodGet,
		requestData{
			relPath:     adminAPIPrefix + "/list-config-history-kv",
			queryValues: v,
		})
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	data, err := DecryptData(adm.getSecretKey(), resp.Body)
	if err != nil {
		return nil, err
	}

	var chEntries []ConfigHistoryEntry
	if err = json.Unmarshal(data, &chEntries); err != nil {
		return chEntries, err
	}

	return chEntries, nil
}
