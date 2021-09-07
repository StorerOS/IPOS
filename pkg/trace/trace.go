package trace

import (
	"net/http"
	"time"
)

type Info struct {
	NodeName  string       `json:"nodename"`
	FuncName  string       `json:"funcname"`
	ReqInfo   RequestInfo  `json:"request"`
	RespInfo  ResponseInfo `json:"response"`
	CallStats CallStats    `json:"stats"`
}

type CallStats struct {
	InputBytes      int           `json:"inputbytes"`
	OutputBytes     int           `json:"outputbytes"`
	Latency         time.Duration `json:"latency"`
	TimeToFirstByte time.Duration `json:"timetofirstbyte"`
}

type RequestInfo struct {
	Time     time.Time   `json:"time"`
	Method   string      `json:"method"`
	Path     string      `json:"path,omitempty"`
	RawQuery string      `json:"rawquery,omitempty"`
	Headers  http.Header `json:"headers,omitempty"`
	Body     []byte      `json:"body,omitempty"`
	Client   string      `json:"client"`
}

type ResponseInfo struct {
	Time       time.Time   `json:"time"`
	Headers    http.Header `json:"headers,omitempty"`
	Body       []byte      `json:"body,omitempty"`
	StatusCode int         `json:"statuscode,omitempty"`
}
