package cmd

import (
	"bytes"
	"context"
	"errors"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/hash"
)

var errConfigNotFound = errors.New("config file not found")

func readConfig(ctx context.Context, objAPI ObjectLayer, configFile string) ([]byte, error) {
	var buffer bytes.Buffer
	if err := objAPI.GetObject(ctx, iposMetaBucket, configFile, 0, -1, &buffer, "", ObjectOptions{}); err != nil {
		if isErrObjectNotFound(err) {
			return nil, errConfigNotFound
		}

		logger.GetReqInfo(ctx).AppendTags("configFile", configFile)
		logger.LogIf(ctx, err)
		return nil, err
	}

	if buffer.Len() == 0 {
		return nil, errConfigNotFound
	}

	return buffer.Bytes(), nil
}

func deleteConfig(ctx context.Context, objAPI ObjectLayer, configFile string) error {
	err := objAPI.DeleteObject(ctx, iposMetaBucket, configFile)
	if err != nil && isErrObjectNotFound(err) {
		return errConfigNotFound
	}
	return err
}

func saveConfig(ctx context.Context, objAPI ObjectLayer, configFile string, data []byte) error {
	hashReader, err := hash.NewReader(bytes.NewReader(data), int64(len(data)), "", getSHA256Hash(data), int64(len(data)), globalCLIContext.StrictS3Compat)
	if err != nil {
		return err
	}

	_, err = objAPI.PutObject(ctx, iposMetaBucket, configFile, NewPutObjReader(hashReader, nil, nil), ObjectOptions{})
	return err
}

func checkConfig(ctx context.Context, objAPI ObjectLayer, configFile string) error {
	if _, err := objAPI.GetObjectInfo(ctx, iposMetaBucket, configFile, ObjectOptions{}); err != nil {
		if isErrObjectNotFound(err) {
			return errConfigNotFound
		}

		logger.GetReqInfo(ctx).AppendTags("configFile", configFile)
		logger.LogIf(ctx, err)
		return err
	}
	return nil
}
