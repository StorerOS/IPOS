package madmin

import (
	"encoding/xml"
	"net/http"
)

type ErrorResponse struct {
	XMLName    xml.Name `xml:"Error" json:"-"`
	Code       string
	Message    string
	BucketName string
	Key        string
	RequestID  string `xml:"RequestId"`
	HostID     string `xml:"HostId"`

	Region string
}

func (e ErrorResponse) Error() string {
	return e.Message
}

const (
	reportIssue = "Please report this issue at http://github.com/storeros/ipos/issues."
)

func httpRespToErrorResponse(resp *http.Response) error {
	if resp == nil {
		msg := "Response is empty. " + reportIssue
		return ErrInvalidArgument(msg)
	}
	var errResp ErrorResponse
	err := jsonDecoder(resp.Body, &errResp)
	if err != nil {
		return ErrorResponse{
			Code:    resp.Status,
			Message: "Failed to parse server response.",
		}
	}
	closeResponse(resp)
	return errResp
}

func ToErrorResponse(err error) ErrorResponse {
	switch err := err.(type) {
	case ErrorResponse:
		return err
	default:
		return ErrorResponse{}
	}
}

func ErrInvalidArgument(message string) error {
	return ErrorResponse{
		Code:      "InvalidArgument",
		Message:   message,
		RequestID: "ipos",
	}
}
