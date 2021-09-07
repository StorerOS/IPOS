package http

import (
	"io"
	"io/ioutil"
)

func DrainBody(respBody io.ReadCloser) {
	if respBody != nil {
		defer respBody.Close()
		io.Copy(ioutil.Discard, respBody)
	}
}
