package openshift

import (
	"errors"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/utils"
	"github.com/xeipuuv/gojsonpointer"
)

// ResourceList is a collection of resources that conform to a filter.
type ResourceList struct {
	Filter *ResourceFilter
	Items  []*ResourceItem
}

// NewTemplateBasedResourceList assembles a ResourceList from an input that is
// treated as coming from a local template (desired state).
func NewTemplateBasedResourceList(filter *ResourceFilter, inputs ...[]byte) (*ResourceList, error) {
	list := &ResourceList{Filter: filter}
	err := list.appendItems("template", "/items", inputs...)
	return list, err
}

// NewPlatformBasedResourceList assembles a ResourceList from an input that is
// treated as coming from an OpenShift cluster (current state).
func NewPlatformBasedResourceList(filter *ResourceFilter, inputs ...[]byte) (*ResourceList, error) {
	list := &ResourceList{Filter: filter}
	err := list.appendItems("platform", "/objects", inputs...)
	return list, err
}

// Length returns the number of items in the resource list
func (l *ResourceList) Length() int {
	return len(l.Items)
}

func (l *ResourceList) getItem(kind string, name string) (*ResourceItem, error) {
	for _, item := range l.Items {
		if item.Kind == kind && item.Name == name {
			return item, nil
		}
	}
	return nil, errors.New("No such item")
}

func (l *ResourceList) appendItems(source, itemsField string, inputs ...[]byte) error {
	for _, input := range inputs {
		if len(input) == 0 {
			cli.DebugMsg("Input config empty")
			continue
		}

		var f interface{}
		err := yaml.Unmarshal(input, &f)
		if err != nil {
			err = utils.DisplaySyntaxError(input, err)
			return err
		}
		m := f.(map[string]interface{})

		p, _ := gojsonpointer.NewJsonPointer(itemsField)
		items, _, err := p.Get(m)
		if err != nil {
			return err
		}
		for _, v := range items.([]interface{}) {
			item, err := NewResourceItem(v.(map[string]interface{}), source)
			if err != nil {
				return err
			}
			if l.Filter.SatisfiedBy(item) {
				l.Items = append(l.Items, item)
			}
		}
	}

	return nil
}
