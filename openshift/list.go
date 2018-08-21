package openshift

import (
	"errors"

	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonpointer"
)

type ResourceList struct {
	Filter *ResourceFilter
	Items  []*ResourceItem
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

func (l *ResourceList) CollectItemsFromTemplateList(input []byte) error {
	return l.appendItemsFromConfig(input, "template")
}

func (l *ResourceList) CollectItemsFromPlatformList(input []byte) error {
	return l.appendItemsFromConfig(input, "platform")
}

func (l *ResourceList) appendItemsFromConfig(input []byte, source string) error {
	if len(input) == 0 {
		return errors.New("Input config empty")
	}

	var f interface{}
	yaml.Unmarshal(input, &f)
	m := f.(map[string]interface{})

	p, _ := gojsonpointer.NewJsonPointer("/items")
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

	return nil
}
