package openshift

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/opendevstack/tailor/utils"
)

var availableKinds = []string{
	"svc",
	"route",
	"dc",
	"bc",
	"is",
	"pvc",
	"template",
	"cm",
	"secret",
	"rolebinding",
	"serviceaccount",
}

type ResourceFilter struct {
	Kinds []string
	Name  string
	Label string
}

// NewResourceFilter returns a filter based on kinds and flags.
// kindArg might be blank, or a list of kinds (e.g. 'pvc,dc') or
// a kind/name combination (e.g. 'dc/foo').
// selectorFlag might be blank or a key and a label, e.g. 'name=foo'.
func NewResourceFilter(kindArg string, selectorFlag string) (*ResourceFilter, error) {
	filter := &ResourceFilter{
		Kinds: []string{},
		Name:  "",
		Label: selectorFlag,
	}

	if len(kindArg) == 0 {
		return filter, nil
	}

	kindArg = strings.ToLower(kindArg)

	if strings.Contains(kindArg, "/") {
		if strings.Contains(kindArg, ",") {
			return nil, errors.New(
				"You cannot target more than one resource name",
			)
		}
		nameParts := strings.Split(kindArg, "/")
		filter.Name = KindMapping[nameParts[0]] + "/" + nameParts[1]
		return filter, nil
	}

	targetedKinds := make(map[string]bool)
	unknownKinds := []string{}
	kinds := strings.Split(kindArg, ",")
	for _, kind := range kinds {
		if _, ok := KindMapping[kind]; !ok {
			unknownKinds = append(unknownKinds, kind)
		} else {
			targetedKinds[KindMapping[kind]] = true
		}
	}

	if len(unknownKinds) > 0 {
		return nil, fmt.Errorf(
			"Unknown resource kinds: %s",
			strings.Join(unknownKinds, ","),
		)
	}

	for kind := range targetedKinds {
		filter.Kinds = append(filter.Kinds, kind)
	}

	sort.Strings(filter.Kinds)

	return filter, nil
}

func (f *ResourceFilter) String() string {
	return fmt.Sprintf("Kind: %s, Name: %s, Label: %s", f.Kinds, f.Name, f.Label)
}

func (f *ResourceFilter) SatisfiedBy(item *ResourceItem) bool {
	if len(f.Name) > 0 && f.Name != item.FullName() {
		return false
	}

	if len(f.Kinds) > 0 && !utils.Includes(f.Kinds, item.Kind) {
		return false
	}

	if len(f.Label) > 0 {
		labels := strings.Split(f.Label, ",")
		for _, label := range labels {
			labelParts := strings.Split(label, "=")
			if _, ok := item.Labels[labelParts[0]]; !ok {
				return false
			} else if item.Labels[labelParts[0]].(string) != labelParts[1] {
				return false
			}
		}
	}

	return true
}

func (f *ResourceFilter) ConvertToTarget() string {
	if len(f.Name) > 0 {
		return f.Name
	}
	kinds := f.Kinds
	if len(kinds) == 0 {
		kinds = availableKinds
	}
	return strings.Join(kinds, ",")
}

func (f *ResourceFilter) ConvertToKinds() string {
	if len(f.Name) > 0 {
		nameParts := strings.Split(f.Name, "/")
		return nameParts[0]
	}
	kinds := f.Kinds
	if len(kinds) == 0 {
		kinds = availableKinds
	}
	return strings.Join(kinds, ",")
}
