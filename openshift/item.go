package openshift

import (
	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonpointer"
	"reflect"
)

var (
	immuntableFields = map[string][]string{
		"Route": []string{
			"/spec/host",
		},
		"PersistentVolumeClaim": []string{
			"/spec/accessModes",
			"/spec/storageClassName",
			"/spec/resources/requests/storage",
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
	if currentItem != nil {
		for k, v := range currentItem.OriginalValues {
			pointer, _ := gojsonpointer.NewJsonPointer(k)
			desiredVal, _, _ := pointer.Get(c)
			currentVal, _, _ := pointer.Get(currentItem.Config)
			if desiredVal == currentVal {
				pointer.Set(c, v)
			}
		}
	}
	y, _ := yaml.Marshal(c)
	return string(y)
}

func (i *ResourceItem) ImmutableFieldsEqual(other *ResourceItem) bool {
	if val, ok := immuntableFields[i.Kind]; ok {
		for _, f := range val {
			pointer, _ := gojsonpointer.NewJsonPointer(f)
			itemVal, _, _ := pointer.Get(i.Config)
			otherVal, _, _ := pointer.Get(other.Config)
			if !reflect.DeepEqual(itemVal, otherVal) {
				return false
			}
		}
	}

	return true
}
