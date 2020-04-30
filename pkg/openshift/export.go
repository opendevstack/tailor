package openshift

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/xeipuuv/gojsonpointer"
)

// ExportAsTemplateFile exports resources in template format.
func ExportAsTemplateFile(filter *ResourceFilter, withAnnotations bool, ocClient cli.OcClientExporter) (string, error) {
	outBytes, err := ocClient.Export(filter.ConvertToKinds(), filter.Label)
	if err != nil {
		return "", fmt.Errorf("Could not export %s resources: %s", filter.String(), err)
	}
	if len(outBytes) == 0 {
		return "", nil
	}

	list, err := NewPlatformBasedResourceList(filter, outBytes)
	if err != nil {
		return "", fmt.Errorf("Could not create resource list from export: %s", err)
	}

	objects := []map[string]interface{}{}
	for _, i := range list.Items {
		if !withAnnotations {
			cli.DebugMsg("Remove annotations from item")
			annotationsPointer, _ := gojsonpointer.NewJsonPointer("/metadata/annotations")
			_, err = annotationsPointer.Delete(i.Config)
			if err != nil {
				cli.DebugMsg("Could not delete annotations from item")
			}
		}
		objects = append(objects, i.Config)
	}

	t := map[string]interface{}{
		"apiVersion": "template.openshift.io/v1",
		"kind":       "Template",
		"objects":    objects,
	}

	b, err := yaml.Marshal(t)
	if err != nil {
		return "", fmt.Errorf(
			"Could not marshal template: %s", err,
		)
	}

	return string(b), err
}