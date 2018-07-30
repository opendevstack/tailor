package openshift

import (
	"errors"
	"github.com/opendevstack/tailor/cli"
	"strconv"
)

type ResourceList struct {
	Filter *ResourceFilter
	Items  []*ResourceItem
}

func NewResourceList(filter *ResourceFilter, config *Config) *ResourceList {
	items := config.ExtractResources(filter)
	l := &ResourceList{Items: items, Filter: filter}
	return l
}

func (l *ResourceList) Length() int {
	return len(l.Items)
}

func (l *ResourceList) GetItem(kind string, name string) (*ResourceItem, error) {
	for _, item := range l.Items {
		if item.Kind == kind && item.Name == name {
			return item, nil
		}
	}
	return nil, errors.New("No such item")
}

func (l *ResourceList) AppendItems(config *Config) {
	items := config.ExtractResources(l.Filter)
	cli.VerboseMsg("Extracted", strconv.Itoa(len(items)), "resources from config")
	l.Items = append(l.Items, items...)
}
