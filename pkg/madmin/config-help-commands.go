package madmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type Help struct {
	SubSys          string  `json:"subSys"`
	Description     string  `json:"description"`
	MultipleTargets bool    `json:"multipleTargets"`
	KeysHelp        HelpKVS `json:"keysHelp"`
}

type HelpKV struct {
	Key             string `json:"key"`
	Description     string `json:"description"`
	Optional        bool   `json:"optional"`
	Type            string `json:"type"`
	MultipleTargets bool   `json:"multipleTargets"`
}

type HelpKVS []HelpKV

func (h Help) Keys() []string {
	var keys []string
	for _, kh := range h.KeysHelp {
		keys = append(keys, kh.Key)
	}
	return keys
}

func (adm *AdminClient) HelpConfigKV(ctx context.Context, subSys, key string, envOnly bool) (Help, error) {
	v := url.Values{}
	v.Set("subSys", subSys)
	v.Set("key", key)
	if envOnly {
		v.Set("env", "")
	}

	reqData := requestData{
		relPath:     adminAPIPrefix + "/help-config-kv",
		queryValues: v,
	}

	resp, err := adm.executeMethod(ctx, http.MethodGet, reqData)
	if err != nil {
		return Help{}, err
	}
	defer closeResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return Help{}, httpRespToErrorResponse(resp)
	}

	var help = Help{}
	d := json.NewDecoder(resp.Body)
	d.DisallowUnknownFields()
	if err = d.Decode(&help); err != nil {
		return help, err
	}

	return help, nil
}
