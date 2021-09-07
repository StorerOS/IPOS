package set

import (
	"fmt"
	"sort"

	jsoniter "github.com/json-iterator/go"
)

type StringSet map[string]struct{}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (set StringSet) ToSlice() []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (set StringSet) IsEmpty() bool {
	return len(set) == 0
}

func (set StringSet) Add(s string) {
	set[s] = struct{}{}
}

func (set StringSet) Remove(s string) {
	delete(set, s)
}

func (set StringSet) Contains(s string) bool {
	_, ok := set[s]
	return ok
}

func (set StringSet) FuncMatch(matchFn func(string, string) bool, matchString string) StringSet {
	nset := NewStringSet()
	for k := range set {
		if matchFn(k, matchString) {
			nset.Add(k)
		}
	}
	return nset
}

func (set StringSet) ApplyFunc(applyFn func(string) string) StringSet {
	nset := NewStringSet()
	for k := range set {
		nset.Add(applyFn(k))
	}
	return nset
}

func (set StringSet) Equals(sset StringSet) bool {
	if len(set) != len(sset) {
		return false
	}

	for k := range set {
		if _, ok := sset[k]; !ok {
			return false
		}
	}

	return true
}

func (set StringSet) Intersection(sset StringSet) StringSet {
	nset := NewStringSet()
	for k := range set {
		if _, ok := sset[k]; ok {
			nset.Add(k)
		}
	}

	return nset
}

func (set StringSet) Difference(sset StringSet) StringSet {
	nset := NewStringSet()
	for k := range set {
		if _, ok := sset[k]; !ok {
			nset.Add(k)
		}
	}

	return nset
}

func (set StringSet) Union(sset StringSet) StringSet {
	nset := NewStringSet()
	for k := range set {
		nset.Add(k)
	}

	for k := range sset {
		nset.Add(k)
	}

	return nset
}

func (set StringSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(set.ToSlice())
}

func (set *StringSet) UnmarshalJSON(data []byte) error {
	sl := []string{}
	var err error
	if err = json.Unmarshal(data, &sl); err == nil {
		*set = make(StringSet)
		for _, s := range sl {
			set.Add(s)
		}
	} else {
		var s string
		if err = json.Unmarshal(data, &s); err == nil {
			*set = make(StringSet)
			set.Add(s)
		}
	}

	return err
}

func (set StringSet) String() string {
	return fmt.Sprintf("%s", set.ToSlice())
}

func NewStringSet() StringSet {
	return make(StringSet)
}

func CreateStringSet(sl ...string) StringSet {
	set := make(StringSet)
	for _, k := range sl {
		set.Add(k)
	}
	return set
}

func CopyStringSet(set StringSet) StringSet {
	nset := NewStringSet()
	for k, v := range set {
		nset[k] = v
	}
	return nset
}
