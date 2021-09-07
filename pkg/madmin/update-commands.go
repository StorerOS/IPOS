package madmin

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

type ServerUpdateStatus struct {
	CurrentVersion string `json:"currentVersion"`
	UpdatedVersion string `json:"updatedVersion"`
}

func (adm *AdminClient) ServerUpdate(ctx context.Context, updateURL string) (us ServerUpdateStatus, err error) {
	queryValues := url.Values{}
	queryValues.Set("updateURL", updateURL)

	resp, err := adm.executeMethod(ctx,
		http.MethodPost, requestData{
			relPath:     adminAPIPrefix + "/update",
			queryValues: queryValues,
		},
	)
	defer closeResponse(resp)
	if err != nil {
		return us, err
	}

	if resp.StatusCode != http.StatusOK {
		return us, httpRespToErrorResponse(resp)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return us, err
	}
	err = json.Unmarshal(buf, &us)
	return us, err
}
