package cmd

import (
	"encoding/base64"
	"net/url"
	"strconv"
)

func getListObjectsV1Args(values url.Values) (prefix, marker, delimiter string, maxkeys int, encodingType string, errCode APIErrorCode) {
	errCode = ErrNone

	if values.Get("max-keys") != "" {
		var err error
		if maxkeys, err = strconv.Atoi(values.Get("max-keys")); err != nil {
			errCode = ErrInvalidMaxKeys
			return
		}
	} else {
		maxkeys = maxObjectList
	}

	prefix = values.Get("prefix")
	marker = values.Get("marker")
	delimiter = values.Get("delimiter")
	encodingType = values.Get("encoding-type")
	return
}

func getListObjectsV2Args(values url.Values) (prefix, token, startAfter, delimiter string, fetchOwner bool, maxkeys int, encodingType string, errCode APIErrorCode) {
	errCode = ErrNone

	if val, ok := values["continuation-token"]; ok {
		if len(val[0]) == 0 {
			errCode = ErrIncorrectContinuationToken
			return
		}
	}

	if values.Get("max-keys") != "" {
		var err error
		if maxkeys, err = strconv.Atoi(values.Get("max-keys")); err != nil {
			errCode = ErrInvalidMaxKeys
			return
		}
	} else {
		maxkeys = maxObjectList
	}

	prefix = values.Get("prefix")
	startAfter = values.Get("start-after")
	delimiter = values.Get("delimiter")
	fetchOwner = values.Get("fetch-owner") == "true"
	encodingType = values.Get("encoding-type")

	if token = values.Get("continuation-token"); token != "" {
		decodedToken, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			errCode = ErrIncorrectContinuationToken
			return
		}
		token = string(decodedToken)
	}
	return
}
