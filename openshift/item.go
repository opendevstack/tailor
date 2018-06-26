package openshift

import (
	"github.com/ghodss/yaml"
)

type ResourceItem struct {
	Kind    string
	Name    string
	Labels  map[string]interface{}
	Pointer string
	Config  interface{}
}

func (i *ResourceItem) FullName() string {
	return i.Kind + "/" + i.Name
}

func (i *ResourceItem) YamlConfig() string {
	y, _ := yaml.Marshal(i.Config)
	return string(y)
}
