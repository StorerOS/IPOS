package log

import "strings"

type Args struct {
	Bucket   string            `json:"bucket,omitempty"`
	Object   string            `json:"object,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Trace struct {
	Message   string            `json:"message,omitempty"`
	Source    []string          `json:"source,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

type API struct {
	Name string `json:"name,omitempty"`
	Args *Args  `json:"args,omitempty"`
}

type Entry struct {
	DeploymentID string `json:"deploymentid,omitempty"`
	Level        string `json:"level"`
	LogKind      string `json:"errKind"`
	Time         string `json:"time"`
	API          *API   `json:"api,omitempty"`
	RemoteHost   string `json:"remotehost,omitempty"`
	Host         string `json:"host,omitempty"`
	RequestID    string `json:"requestID,omitempty"`
	UserAgent    string `json:"userAgent,omitempty"`
	Message      string `json:"message,omitempty"`
	Trace        *Trace `json:"error,omitempty"`
}

type Info struct {
	Entry
	ConsoleMsg string
	NodeName   string `json:"node"`
	Err        error  `json:"-"`
}

func (l Info) SendLog(node, logKind string) bool {
	nodeFltr := (node == "" || strings.EqualFold(node, l.NodeName))
	typeFltr := strings.EqualFold(logKind, "all") || strings.EqualFold(l.LogKind, logKind)
	return nodeFltr && typeFltr
}
