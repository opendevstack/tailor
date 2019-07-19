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
	Kinds          []string
	Name           string
	Label          string
	ExcludedKinds  []string
	ExcludedNames  []string
	ExcludedLabels []string
}

// NewResourceFilter returns a filter based on kinds and flags.
// kindArg might be blank, or a list of kinds (e.g. 'pvc,dc') or
// a kind/name combination (e.g. 'dc/foo').
// selectorFlag might be blank or a key and a label, e.g. 'name=foo'.
func NewResourceFilter(kindArg string, selectorFlag string, excludeFlag string) (*ResourceFilter, error) {
	filter := &ResourceFilter{
		Kinds: []string{},
		Name:  "",
		Label: selectorFlag,
	}

	if len(kindArg) > 0 {
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
	}

	if len(excludeFlag) > 0 {
		unknownKinds := []string{}
		excludes := strings.Split(excludeFlag, ",")
		for _, v := range excludes {
			v = strings.ToLower(v)
			if strings.Contains(v, "/") { // Name
				nameParts := strings.Split(v, "/")
				filter.ExcludedNames = append(filter.ExcludedNames, KindMapping[nameParts[0]]+"/"+nameParts[1])
			} else if strings.Contains(v, "=") { // Label
				filter.ExcludedLabels = append(filter.ExcludedLabels, v)
			} else { // Kind
				if _, ok := KindMapping[v]; !ok {
					unknownKinds = append(unknownKinds, v)
				} else {
					filter.ExcludedKinds = append(filter.ExcludedKinds, KindMapping[v])
				}
			}
		}

		if len(unknownKinds) > 0 {
			return nil, fmt.Errorf(
				"Unknown excluded resource kinds: %s",
				strings.Join(unknownKinds, ","),
			)
		}
	}

	return filter, nil
}

func (f *ResourceFilter) String() string {
	return fmt.Sprintf("Kinds: %s, Name: %s, Label: %s, ExcludedKinds: %s, ExcludedNames: %s, ExcludedLabels: %s", f.Kinds, f.Name, f.Label, f.ExcludedKinds, f.ExcludedNames, f.ExcludedLabels)
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
			if !item.HasLabel(label) {
				return false
			}
		}
	}

	if len(f.ExcludedNames) > 0 {
		if utils.Includes(f.ExcludedNames, item.FullName()) {
			return false
		}
	}

	if len(f.ExcludedKinds) > 0 {
		if utils.Includes(f.ExcludedKinds, item.Kind) {
			return false
		}
	}

	if len(f.ExcludedLabels) > 0 {
		for _, el := range f.ExcludedLabels {
			if item.HasLabel(el) {
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
