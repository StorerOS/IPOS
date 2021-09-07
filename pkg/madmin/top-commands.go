package madmin

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

type LockEntry struct {
	Timestamp  time.Time `json:"time"`
	Resource   string    `json:"resource"`
	Type       string    `json:"type"`
	Source     string    `json:"source"`
	ServerList []string  `json:"serverlist"`
	Owner      string    `json:"owner"`
	ID         string    `json:"id"`
}

type LockEntries []LockEntry

func (l LockEntries) Len() int {
	return len(l)
}

func (l LockEntries) Less(i, j int) bool {
	return l[i].Timestamp.Before(l[j].Timestamp)
}

func (l LockEntries) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (adm *AdminClient) TopLocks(ctx context.Context) (LockEntries, error) {
	resp, err := adm.executeMethod(ctx,
		http.MethodGet,
		requestData{relPath: adminAPIPrefix + "/top/locks"},
	)
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return LockEntries{}, err
	}

	var lockEntries LockEntries
	err = json.Unmarshal(response, &lockEntries)
	return lockEntries, err
}
