package console

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/cmd/ipos/logger/message/log"
	"github.com/storeros/ipos/pkg/color"
	"github.com/storeros/ipos/pkg/console"
)

type Target struct{}

func (c *Target) Send(e interface{}, logKind string) error {
	entry, ok := e.(log.Entry)
	if !ok {
		return fmt.Errorf("Uexpected log entry structure %#v", e)
	}
	if logger.IsJSON() {
		logJSON, err := json.Marshal(&entry)
		if err != nil {
			return err
		}
		fmt.Println(string(logJSON))
		return nil
	}

	traceLength := len(entry.Trace.Source)
	trace := make([]string, traceLength)

	for i, element := range entry.Trace.Source {
		trace[i] = fmt.Sprintf("%8v: %s", traceLength-i, element)
	}

	tagString := ""
	for key, value := range entry.Trace.Variables {
		if value != "" {
			if tagString != "" {
				tagString += ", "
			}
			tagString += key + "=" + value
		}
	}

	apiString := "API: " + entry.API.Name + "("
	if entry.API.Args != nil && entry.API.Args.Bucket != "" {
		apiString = apiString + "bucket=" + entry.API.Args.Bucket
	}
	if entry.API.Args != nil && entry.API.Args.Object != "" {
		apiString = apiString + ", object=" + entry.API.Args.Object
	}
	apiString += ")"
	timeString := "Time: " + time.Now().Format(logger.TimeFormat)

	var deploymentID string
	if entry.DeploymentID != "" {
		deploymentID = "\nDeploymentID: " + entry.DeploymentID
	}

	var requestID string
	if entry.RequestID != "" {
		requestID = "\nRequestID: " + entry.RequestID
	}

	var remoteHost string
	if entry.RemoteHost != "" {
		remoteHost = "\nRemoteHost: " + entry.RemoteHost
	}

	var host string
	if entry.Host != "" {
		host = "\nHost: " + entry.Host
	}

	var userAgent string
	if entry.UserAgent != "" {
		userAgent = "\nUserAgent: " + entry.UserAgent
	}

	if len(entry.Trace.Variables) > 0 {
		tagString = "\n       " + tagString
	}

	var msg = color.FgRed(color.Bold(entry.Trace.Message))
	var output = fmt.Sprintf("\n%s\n%s%s%s%s%s%s\nError: %s%s\n%s",
		apiString, timeString, deploymentID, requestID, remoteHost, host, userAgent,
		msg, tagString, strings.Join(trace, "\n"))

	console.Println(output)
	return nil
}

func New() *Target {
	return &Target{}
}
