package iampolicy

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/storeros/ipos/pkg/bucket/policy/condition"
	"github.com/storeros/ipos/pkg/wildcard"
)

const ResourceARNPrefix = "arn:aws:s3:::"

type Resource struct {
	BucketName string
	Pattern    string
}

func (r Resource) isBucketPattern() bool {
	return !strings.Contains(r.Pattern, "/") || r.Pattern == "*"
}

func (r Resource) isObjectPattern() bool {
	return strings.Contains(r.Pattern, "/") || strings.Contains(r.BucketName, "*") || r.Pattern == "*/*"
}

func (r Resource) IsValid() bool {
	return r.Pattern != ""
}

func (r Resource) Match(resource string, conditionValues map[string][]string) bool {
	pattern := r.Pattern
	for _, key := range condition.CommonKeys {
		if rvalues, ok := conditionValues[key.Name()]; ok && rvalues[0] != "" {
			pattern = strings.Replace(pattern, key.VarName(), rvalues[0], -1)
		}
	}
	if path.Clean(resource) == pattern {
		return true
	}
	return wildcard.Match(pattern, resource)
}

func (r Resource) MarshalJSON() ([]byte, error) {
	if !r.IsValid() {
		return nil, Errorf("invalid resource %v", r)
	}

	return json.Marshal(r.String())
}

func (r Resource) String() string {
	return ResourceARNPrefix + r.Pattern
}

func (r *Resource) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsedResource, err := parseResource(s)
	if err != nil {
		return err
	}

	*r = parsedResource

	return nil
}

func (r Resource) Validate() error {
	if !r.IsValid() {
		return Errorf("invalid resource")
	}
	return nil
}

func parseResource(s string) (Resource, error) {
	if !strings.HasPrefix(s, ResourceARNPrefix) {
		return Resource{}, Errorf("invalid resource '%v'", s)
	}

	pattern := strings.TrimPrefix(s, ResourceARNPrefix)
	tokens := strings.SplitN(pattern, "/", 2)
	bucketName := tokens[0]
	if bucketName == "" {
		return Resource{}, Errorf("invalid resource format '%v'", s)
	}

	return Resource{
		BucketName: bucketName,
		Pattern:    pattern,
	}, nil
}

func NewResource(bucketName, keyName string) Resource {
	pattern := bucketName
	if keyName != "" {
		if !strings.HasPrefix(keyName, "/") {
			pattern += "/"
		}

		pattern += keyName
	}

	return Resource{
		BucketName: bucketName,
		Pattern:    pattern,
	}
}
