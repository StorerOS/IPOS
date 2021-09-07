package madmin

import (
	"context"
	"net/http"
	"net/url"
)

func (adm *AdminClient) DelConfigKV(ctx context.Context, k string) (err error) {
	econfigBytes, err := EncryptData(adm.getSecretKey(), []byte(k))
	if err != nil {
		return err
	}

	reqData := requestData{
		relPath: adminAPIPrefix + "/del-config-kv",
		content: econfigBytes,
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

func (adm *AdminClient) SetConfigKV(ctx context.Context, kv string) (err error) {
	econfigBytes, err := EncryptData(adm.getSecretKey(), []byte(kv))
	if err != nil {
		return err
	}

	reqData := requestData{
		relPath: adminAPIPrefix + "/set-config-kv",
		content: econfigBytes,
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

func (adm *AdminClient) GetConfigKV(ctx context.Context, key string) ([]byte, error) {
	v := url.Values{}
	v.Set("key", key)

	resp, err := adm.executeMethod(ctx,
		http.MethodGet,
		requestData{
			relPath:     adminAPIPrefix + "/get-config-kv",
			queryValues: v,
		})
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	defer closeResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	return DecryptData(adm.getSecretKey(), resp.Body)
}
