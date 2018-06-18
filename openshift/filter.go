package openshift

import (
	"fmt"
	"github.com/michaelsauter/ocdiff/utils"
	"strings"
)

type ResourceFilter struct {
	Kind    string
	Names   []string
	Label   string
}

func (f *ResourceFilter) String() string {
	return fmt.Sprintf("Kind: %s, Names: %s, Label: %s", f.Kind, f.Names, f.Label)
}

func (f *ResourceFilter) SatisfiedBy(item *ResourceItem) bool {
	if (len(f.Kind) > 0 && f.Kind != item.Kind) {
		return false
	}
	if (len(f.Names) > 0) && !utils.Includes(f.Names, item.Name) {
		return false
	}
	if (len(f.Label) > 0) {
		labelParts := strings.Split(f.Label, "=")
		if _, ok := item.Labels[labelParts[0]]; !ok {
			return false
		} else if item.Labels[labelParts[0]].(string) != labelParts[1] {
			return false
		}
	}
	return true
}

