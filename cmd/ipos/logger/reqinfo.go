package logger

import (
	"context"
	"fmt"
	"sync"
)

type contextKeyType string

const contextLogKey = contextKeyType("iposlog")

type KeyVal struct {
	Key string
	Val string
}

type ReqInfo struct {
	RemoteHost   string
	Host         string
	UserAgent    string
	DeploymentID string
	RequestID    string
	API          string
	BucketName   string
	ObjectName   string
	tags         []KeyVal
	sync.RWMutex
}

func NewReqInfo(remoteHost, userAgent, deploymentID, requestID, api, bucket, object string) *ReqInfo {
	req := ReqInfo{}
	req.RemoteHost = remoteHost
	req.UserAgent = userAgent
	req.API = api
	req.DeploymentID = deploymentID
	req.RequestID = requestID
	req.BucketName = bucket
	req.ObjectName = object
	return &req
}

func (r *ReqInfo) AppendTags(key string, val string) *ReqInfo {
	if r == nil {
		return nil
	}
	r.Lock()
	defer r.Unlock()
	r.tags = append(r.tags, KeyVal{key, val})
	return r
}

func (r *ReqInfo) SetTags(key string, val string) *ReqInfo {
	if r == nil {
		return nil
	}
	r.Lock()
	defer r.Unlock()
	var updated bool
	for _, tag := range r.tags {
		if tag.Key == key {
			tag.Val = val
			updated = true
			break
		}
	}
	if !updated {
		r.tags = append(r.tags, KeyVal{key, val})
	}
	return r
}

func (r *ReqInfo) GetTags() []KeyVal {
	if r == nil {
		return nil
	}
	r.RLock()
	defer r.RUnlock()
	return append([]KeyVal(nil), r.tags...)
}

func SetReqInfo(ctx context.Context, req *ReqInfo) context.Context {
	if ctx == nil {
		LogIf(context.Background(), fmt.Errorf("context is nil"))
		return nil
	}
	return context.WithValue(ctx, contextLogKey, req)
}

func GetReqInfo(ctx context.Context) *ReqInfo {
	if ctx != nil {
		r, ok := ctx.Value(contextLogKey).(*ReqInfo)
		if ok {
			return r
		}
		r = &ReqInfo{}
		SetReqInfo(ctx, r)
		return r
	}
	return nil
}
