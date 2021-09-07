package madmin

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

func (adm *AdminClient) GetConfig(ctx context.Context) ([]byte, error) {
	resp, err := adm.executeMethod(ctx,
		http.MethodGet,
		requestData{relPath: adminAPIPrefix + "/config"})
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}

	return DecryptData(adm.getSecretKey(), resp.Body)
}

func (adm *AdminClient) SetConfig(ctx context.Context, config io.Reader) (err error) {
	const maxConfigJSONSize = 256 * 1024

	configBuf := make([]byte, maxConfigJSONSize+1)
	n, err := io.ReadFull(config, configBuf)
	if err == nil {
		return bytes.ErrTooLarge
	}
	if err != io.ErrUnexpectedEOF {
		return err
	}
	configBytes := configBuf[:n]
	econfigBytes, err := EncryptData(adm.getSecretKey(), configBytes)
	if err != nil {
		return err
	}

	reqData := requestData{
		relPath: adminAPIPrefix + "/config",
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
