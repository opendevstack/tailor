package openshift

import (
	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonpointer"
)

var (
	immuntableFields = map[string][]string{
		"Route": []string{
			"/spec/host",
		},
	}
)

type ResourceItem struct {
	Kind           string
	Name           string
	Labels         map[string]interface{}
	Pointer        string
	Config         interface{}
	OriginalValues map[string]interface{}
}

func (i *ResourceItem) FullName() string {
	return i.Kind + "/" + i.Name
}

func (i *ResourceItem) YamlConfig() string {
	y, _ := yaml.Marshal(i.Config)
	return string(y)
}

func (i *ResourceItem) DesiredConfig(currentItem *ResourceItem) string {
	c := i.Config
	for k, v := range currentItem.OriginalValues {
		pointer, _ := gojsonpointer.NewJsonPointer(k)
		desiredVal, _, _ := pointer.Get(c)
		currentVal, _, _ := pointer.Get(currentItem.Config)
		if desiredVal == currentVal {
			pointer.Set(c, v)
		}
	}
	y, _ := yaml.Marshal(c)
	return string(y)
}

func (i *ResourceItem) GetField(pointer string) (string, error) {
	p, _ := gojsonpointer.NewJsonPointer(pointer)
	val, _, err := p.Get(i.Config)
	return val.(string), err
}

func (i *ResourceItem) ImmutableFieldsEqual(other *ResourceItem) bool {
	if val, ok := immuntableFields[i.Kind]; ok {
		for _, f := range val {
			itemVal, itemErr := i.GetField(f)
			otherVal, otherErr := other.GetField(f)
			if (itemErr == nil && otherErr != nil) || (itemErr != nil && otherErr == nil) || itemVal != otherVal {
				return false
			}
		}
	}

	return true
}
