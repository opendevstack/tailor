package openshift

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/pkg/cli"
)

var (
	trimAnnotationsDefault = []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		"openshift.io/image.dockerRepositoryCheck",
	}
)

// ExportAsTemplateFile exports resources in template format.
func ExportAsTemplateFile(filter *ResourceFilter, withAnnotations bool, namespace string, withHardcodedNamespace bool, trimAnnotations []string, ocClient cli.OcClientExporter) (string, error) {
	outBytes, err := ocClient.Export(filter.ConvertToKinds(), filter.Label)
	if err != nil {
		return "", fmt.Errorf("Could not export %s resources: %s", filter.String(), err)
	}
	if len(outBytes) == 0 {
		return "", nil
	}

	if !withHardcodedNamespace {
		outBytes = bytes.Replace(outBytes, []byte(namespace), []byte("${TAILOR_NAMESPACE}"), -1)
	}

	list, err := NewPlatformBasedResourceList(filter, outBytes)
	if err != nil {
		return "", fmt.Errorf("Could not create resource list from export: %s", err)
	}

	objects := []map[string]interface{}{}
	for _, i := range list.Items {
		if withAnnotations {
			cli.DebugMsg("All annotations will be kept in template item")
		} else {
			trimAnnotations = append(trimAnnotations, trimAnnotationsDefault...)
			cli.DebugMsg("Trim annotations from template item")
			for ia := range i.Annotations {
				for _, ta := range trimAnnotations {
					if strings.HasSuffix(ta, "/") && strings.HasPrefix(ia, ta) {
						i.removeAnnotion(ia)
					} else if ta == ia {
						i.removeAnnotion(ia)
					}
				}
			}
		}
		objects = append(objects, i.Config)
	}

	t := map[string]interface{}{
		"apiVersion": "template.openshift.io/v1",
		"kind":       "Template",
		"objects":    objects,
	}

	if !withHardcodedNamespace {
		parameters := []map[string]interface{}{
			{
				"name":     "TAILOR_NAMESPACE",
				"required": true,
			},
		}
		t["parameters"] = parameters
	}

	b, err := yaml.Marshal(t)
	if err != nil {
		return "", fmt.Errorf(
			"Could not marshal template: %s", err,
		)
	}

	return string(b), err
}
