package madmin

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KVS []KV

func (kvs KVS) Empty() bool {
	return len(kvs) == 0
}

func (kvs *KVS) Set(key, value string) {
	for i, kv := range *kvs {
		if kv.Key == key {
			(*kvs)[i] = KV{
				Key:   key,
				Value: value,
			}
			return
		}
	}
	*kvs = append(*kvs, KV{
		Key:   key,
		Value: value,
	})
}

func (kvs KVS) Get(key string) string {
	v, ok := kvs.Lookup(key)
	if ok {
		return v
	}
	return ""
}

func (kvs KVS) Lookup(key string) (string, bool) {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

type Target struct {
	SubSystem string `json:"subSys"`
	KVS       KVS    `json:"kvs"`
}

const (
	EnableKey  = "enable"
	CommentKey = "comment"

	EnableOn  = "on"
	EnableOff = "off"
)

func HasSpace(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

const (
	SubSystemSeparator = `:`
	KvSeparator        = `=`
	KvComment          = `#`
	KvSpaceSeparator   = ` `
	KvNewline          = "\n"
	KvDoubleQuote      = `"`
	KvSingleQuote      = `'`

	Default = `_`
)

func SanitizeValue(v string) string {
	v = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(v), KvDoubleQuote), KvDoubleQuote)
	return strings.TrimSuffix(strings.TrimPrefix(v, KvSingleQuote), KvSingleQuote)
}

func KvFields(input string, keys []string) []string {
	var valueIndexes []int
	for _, key := range keys {
		i := strings.Index(input, key+KvSeparator)
		if i == -1 {
			continue
		}
		valueIndexes = append(valueIndexes, i)
	}

	sort.Ints(valueIndexes)
	var fields = make([]string, len(valueIndexes))
	for i := range valueIndexes {
		j := i + 1
		if j < len(valueIndexes) {
			fields[i] = strings.TrimSpace(input[valueIndexes[i]:valueIndexes[j]])
		} else {
			fields[i] = strings.TrimSpace(input[valueIndexes[i]:])
		}
	}
	return fields
}

func ParseTarget(s string, help Help) (*Target, error) {
	inputs := strings.SplitN(s, KvSpaceSeparator, 2)
	if len(inputs) <= 1 {
		return nil, fmt.Errorf("invalid number of arguments '%s'", s)
	}

	subSystemValue := strings.SplitN(inputs[0], SubSystemSeparator, 2)
	if len(subSystemValue) == 0 {
		return nil, fmt.Errorf("invalid number of arguments %s", s)
	}

	if help.SubSys != subSystemValue[0] {
		return nil, fmt.Errorf("unknown sub-system %s", subSystemValue[0])
	}

	var kvs = KVS{}
	var prevK string
	for _, v := range KvFields(inputs[1], help.Keys()) {
		kv := strings.SplitN(v, KvSeparator, 2)
		if len(kv) == 0 {
			continue
		}
		if len(kv) == 1 && prevK != "" {
			value := strings.Join([]string{
				kvs.Get(prevK),
				SanitizeValue(kv[0]),
			}, KvSpaceSeparator)
			kvs.Set(prevK, value)
			continue
		}
		if len(kv) == 2 {
			prevK = kv[0]
			kvs.Set(prevK, SanitizeValue(kv[1]))
			continue
		}
		return nil, fmt.Errorf("value for key '%s' cannot be empty", kv[0])
	}

	return &Target{
		SubSystem: inputs[0],
		KVS:       kvs,
	}, nil
}

func ParseSubSysTarget(buf []byte, help Help) (target *Target, err error) {
	bio := bufio.NewScanner(bytes.NewReader(buf))
	if bio.Scan() {
		return ParseTarget(bio.Text(), help)
	}
	return nil, bio.Err()
}
