package policy

import "github.com/storeros/ipos/pkg/set"

type ConditionKeyMap map[string]set.StringSet

func (ckm ConditionKeyMap) Add(key string, value set.StringSet) {
	if v, ok := ckm[key]; ok {
		ckm[key] = v.Union(value)
	} else {
		ckm[key] = set.CopyStringSet(value)
	}
}

func (ckm ConditionKeyMap) Remove(key string, value set.StringSet) {
	if v, ok := ckm[key]; ok {
		if value != nil {
			ckm[key] = v.Difference(value)
		}

		if ckm[key].IsEmpty() {
			delete(ckm, key)
		}
	}
}

func (ckm ConditionKeyMap) RemoveKey(key string) {
	if _, ok := ckm[key]; ok {
		delete(ckm, key)
	}
}

func CopyConditionKeyMap(condKeyMap ConditionKeyMap) ConditionKeyMap {
	out := make(ConditionKeyMap)

	for k, v := range condKeyMap {
		out[k] = set.CopyStringSet(v)
	}

	return out
}

func mergeConditionKeyMap(condKeyMap1 ConditionKeyMap, condKeyMap2 ConditionKeyMap) ConditionKeyMap {
	out := CopyConditionKeyMap(condKeyMap1)

	for k, v := range condKeyMap2 {
		if ev, ok := out[k]; ok {
			out[k] = ev.Union(v)
		} else {
			out[k] = set.CopyStringSet(v)
		}
	}

	return out
}

type ConditionMap map[string]ConditionKeyMap

func (cond ConditionMap) Add(condKey string, condKeyMap ConditionKeyMap) {
	if v, ok := cond[condKey]; ok {
		cond[condKey] = mergeConditionKeyMap(v, condKeyMap)
	} else {
		cond[condKey] = CopyConditionKeyMap(condKeyMap)
	}
}

func (cond ConditionMap) Remove(condKey string) {
	if _, ok := cond[condKey]; ok {
		delete(cond, condKey)
	}
}

func mergeConditionMap(condMap1 ConditionMap, condMap2 ConditionMap) ConditionMap {
	out := make(ConditionMap)

	for k, v := range condMap1 {
		out[k] = CopyConditionKeyMap(v)
	}

	for k, v := range condMap2 {
		if ev, ok := out[k]; ok {
			out[k] = mergeConditionKeyMap(ev, v)
		} else {
			out[k] = CopyConditionKeyMap(v)
		}
	}

	return out
}
