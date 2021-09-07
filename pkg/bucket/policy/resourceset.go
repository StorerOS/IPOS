package policy

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/storeros/ipos/pkg/set"
)

type ResourceSet map[Resource]struct{}

func (resourceSet ResourceSet) bucketResourceExists() bool {
	for resource := range resourceSet {
		if resource.isBucketPattern() {
			return true
		}
	}

	return false
}

func (resourceSet ResourceSet) objectResourceExists() bool {
	for resource := range resourceSet {
		if resource.isObjectPattern() {
			return true
		}
	}

	return false
}

func (resourceSet ResourceSet) Add(resource Resource) {
	resourceSet[resource] = struct{}{}
}

func (resourceSet ResourceSet) Equals(sresourceSet ResourceSet) bool {
	if len(resourceSet) != len(sresourceSet) {
		return false
	}

	for k := range resourceSet {
		if _, ok := sresourceSet[k]; !ok {
			return false
		}
	}

	return true
}

func (resourceSet ResourceSet) Intersection(sset ResourceSet) ResourceSet {
	nset := NewResourceSet()
	for k := range resourceSet {
		if _, ok := sset[k]; ok {
			nset.Add(k)
		}
	}

	return nset
}

func (resourceSet ResourceSet) MarshalJSON() ([]byte, error) {
	if len(resourceSet) == 0 {
		return nil, Errorf("empty resources not allowed")
	}

	resources := []Resource{}
	for resource := range resourceSet {
		resources = append(resources, resource)
	}

	return json.Marshal(resources)
}

func (resourceSet ResourceSet) Match(resource string, conditionValues map[string][]string) bool {
	for r := range resourceSet {
		if r.Match(resource, conditionValues) {
			return true
		}
	}

	return false
}

func (resourceSet ResourceSet) String() string {
	resources := []string{}
	for resource := range resourceSet {
		resources = append(resources, resource.String())
	}
	sort.Strings(resources)

	return fmt.Sprintf("%v", resources)
}

func (resourceSet *ResourceSet) UnmarshalJSON(data []byte) error {
	var sset set.StringSet
	if err := json.Unmarshal(data, &sset); err != nil {
		return err
	}

	*resourceSet = make(ResourceSet)
	for _, s := range sset.ToSlice() {
		resource, err := parseResource(s)
		if err != nil {
			return err
		}

		if _, found := (*resourceSet)[resource]; found {
			return Errorf("duplicate resource '%v' found", s)
		}

		resourceSet.Add(resource)
	}

	return nil
}

func (resourceSet ResourceSet) Validate(bucketName string) error {
	for resource := range resourceSet {
		if err := resource.Validate(bucketName); err != nil {
			return err
		}
	}

	return nil
}

func NewResourceSet(resources ...Resource) ResourceSet {
	resourceSet := make(ResourceSet)
	for _, resource := range resources {
		resourceSet.Add(resource)
	}

	return resourceSet
}
