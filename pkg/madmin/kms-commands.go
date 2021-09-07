package madmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

func (adm *AdminClient) GetKeyStatus(ctx context.Context, keyID string) (*KMSKeyStatus, error) {
	qv := url.Values{}
	qv.Set("key-id", keyID)
	reqData := requestData{
		relPath:     adminAPIPrefix + "/kms/key/status",
		queryValues: qv,
	}

	resp, err := adm.executeMethod(ctx, http.MethodGet, reqData)
	if err != nil {
		return nil, err
	}
	defer closeResponse(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}
	var keyInfo KMSKeyStatus
	if err = json.NewDecoder(resp.Body).Decode(&keyInfo); err != nil {
		return nil, err
	}
	return &keyInfo, nil
}

type KMSKeyStatus struct {
	KeyID         string `json:"key-id"`
	EncryptionErr string `json:"encryption-error,omitempty"`
	DecryptionErr string `json:"decryption-error,omitempty"`
}
