package openshift

import (
	"errors"
	"github.com/michaelsauter/ocdiff/cli"
	"strconv"
)

type ResourceList struct {
	Filter *ResourceFilter
	Items  []*ResourceItem
}

func NewResourceList(kind string, config *Config) *ResourceList {
	filter := &ResourceFilter{
		Kind: kind,
	}
	items := config.ExtractResources(filter)
	l := &ResourceList{Items: items, Filter: filter}
	return l
}

func (l *ResourceList) GetItem(name string) (*ResourceItem, error) {
	for _, item := range l.Items {
		if item.Name == name {
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
