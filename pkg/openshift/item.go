package openshift

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/utils"
	"github.com/xeipuuv/gojsonpointer"
)

var (
	annotationsPath             = "/metadata/annotations"
	platformManagedSimpleFields = []string{
		"/groupNames",
		"/imagePullSecrets",
		"/metadata/creationTimestamp",
		"/metadata/generation",
		"/metadata/managedFields",
		"/metadata/namespace",
		"/metadata/resourceVersion",
		"/metadata/selfLink",
		"/metadata/uid",
		"/secrets",
		"/spec/clusterIP",
		"/spec/clusterIPs",
		"/spec/jobTemplate/metadata/creationTimestamp",
		"/spec/jobTemplate/spec/template/metadata/creationTimestamp",
		"/spec/selector/matchLabels/controller-uid",
		"/spec/tags",
		"/spec/template/metadata/creationTimestamp",
		"/spec/template/metadata/labels/controller-uid",
		"/spec/volumeName",
		"/status",
		"/userNames",
	}
	platformManagedRegexFields = []string{
		"^/spec/triggers/[0-9]*/imageChangeParams/lastTriggeredImage",
	}
	immutableFields = map[string][]string{
		"PersistentVolumeClaim": {
			"/spec/accessModes",
			"/spec/storageClassName",
			"/spec/resources/requests/storage",
		},
		"Route": {
			"/spec/host",
		},
		"Secret": {
			"/type",
		},
	}

	KindMapping = map[string]string{
		"svc":                   "Service",
		"service":               "Service",
		"route":                 "Route",
		"dc":                    "DeploymentConfig",
		"deploymentconfig":      "DeploymentConfig",
		"deployment":            "Deployment",
		"bc":                    "BuildConfig",
		"buildconfig":           "BuildConfig",
		"is":                    "ImageStream",
		"imagestream":           "ImageStream",
		"pvc":                   "PersistentVolumeClaim",
		"persistentvolumeclaim": "PersistentVolumeClaim",
		"template":              "Template",
		"cm":                    "ConfigMap",
		"configmap":             "ConfigMap",
		"secret":                "Secret",
		"rolebinding":           "RoleBinding",
		"serviceaccount":        "ServiceAccount",
		"cronjob":               "CronJob",
		"cj":                    "CronJob",
		"job":                   "Job",
		"limitrange":            "LimitRange",
		"resourcequota":         "ResourceQuota",
		"quota":                 "ResourceQuota",
		"hpa":                   "HorizontalPodAutoscaler",
		"statefulset":           "StatefulSet",
	}
)

type ResourceItem struct {
	Source                   string
	Kind                     string
	Name                     string
	Labels                   map[string]interface{}
	Annotations              map[string]interface{}
	Paths                    []string
	Config                   map[string]interface{}
	AnnotationsPresent       bool
	LastAppliedConfiguration map[string]interface{}
	LastAppliedAnnotations   map[string]interface{}
	Comparable               bool
}

func NewResourceItem(m map[string]interface{}, source string) (*ResourceItem, error) {
	item := &ResourceItem{Source: source}
	err := item.parseConfig(m)
	return item, err
}

// FullName returns kind/name, with kind being the long form (e.g. "DeploymentConfig").
func (i *ResourceItem) FullName() string {
	return i.Kind + "/" + i.Name
}

// ShortName returns kind/name, with kind being the shortest possible
// reference of kind (e.g. "dc" for "DeploymentConfig").
func (i *ResourceItem) ShortName() string {
	return kindToShortMapping[i.Kind] + "/" + i.Name
}

func (i *ResourceItem) HasLabel(label string) bool {
	labelParts := strings.Split(label, "=")
	if _, ok := i.Labels[labelParts[0]]; !ok {
		return false
	} else if i.Labels[labelParts[0]].(string) != labelParts[1] {
		return false
	}
	return true
}

func (i *ResourceItem) DesiredConfig() (string, error) {
	y, _ := yaml.Marshal(i.Config)
	return string(y), nil
}

func (i *ResourceItem) YamlConfig() string {
	y, _ := yaml.Marshal(i.Config)
	return string(y)
}

// parseConfig uses the config to initialise an item. The logic is the same
// for template and platform items, with no knowledge of the "other" item - it
// may or may not exist.
func (i *ResourceItem) parseConfig(m map[string]interface{}) error {
	// Extract kind
	kindPointer, _ := gojsonpointer.NewJsonPointer("/kind")
	kind, _, err := kindPointer.Get(m)
	if err != nil {
		return err
	}
	i.Kind = kind.(string)

	// Extract name
	namePointer, _ := gojsonpointer.NewJsonPointer("/metadata/name")
	name, _, noNameErr := namePointer.Get(m)
	if noNameErr == nil {
		i.Name = name.(string)
	} else {
		generateNamePointer, _ := gojsonpointer.NewJsonPointer("/metadata/generateName")
		generateName, _, err := generateNamePointer.Get(m)
		if err != nil {
			return fmt.Errorf("Resource does not have paths /metadata/name or /metadata/generateName: %s", err)
		}
		i.Name = generateName.(string)
	}

	// Determine if item is comparable and therefore relevant for Tailor
	i.Comparable = true
	// Secrets of type "kubernetes.io/dockercfg" and
	// "kubernetes.io/service-account-token" were not returned in "oc export".
	// Those secrets are generated by OpenShift automatically and should not
	// be controlled by Tailor.
	if i.Kind == "Secret" {
		typePointer, _ := gojsonpointer.NewJsonPointer("/type")
		typeVal, _, err := typePointer.Get(m)
		if err != nil {
			return fmt.Errorf("Secret has no field /type: %s", err)
		}
		irrelevantSecrets := []string{
			"kubernetes.io/dockercfg",
			"kubernetes.io/service-account-token",
		}
		if utils.Includes(irrelevantSecrets, typeVal.(string)) {
			i.Comparable = false
			cli.DebugMsg(
				"Removed secret",
				i.Name,
				"of type",
				typeVal.(string),
				"as it cannot be compared properly",
			)
		}
	}

	// Extract labels
	labelsPointer, _ := gojsonpointer.NewJsonPointer("/metadata/labels")
	labels, _, err := labelsPointer.Get(m)
	if err != nil {
		i.Labels = make(map[string]interface{})
	} else {
		i.Labels = labels.(map[string]interface{})
	}

	// Extract annotations
	annotationsPointer, _ := gojsonpointer.NewJsonPointer("/metadata/annotations")
	annotations, _, err := annotationsPointer.Get(m)
	i.Annotations = make(map[string]interface{})
	i.AnnotationsPresent = false
	if err == nil {
		i.AnnotationsPresent = true
		for k, v := range annotations.(map[string]interface{}) {
			i.Annotations[k] = v
		}
	}

	i.LastAppliedConfiguration = make(map[string]interface{})
	i.LastAppliedAnnotations = make(map[string]interface{})

	// kubectl.kubernetes.io/last-applied-configuration
	lastAppliedConfigurationPointer, _ := gojsonpointer.NewJsonPointer("/metadata/annotations/kubectl.kubernetes.io~1last-applied-configuration")
	lastAppliedConfiguration, _, err := lastAppliedConfigurationPointer.Get(m)
	if err == nil {
		s := lastAppliedConfiguration.(string)
		var f interface{}
		err := json.Unmarshal([]byte(s), &f)
		if err != nil {
			return err
		}
		lac := f.(map[string]interface{})
		i.LastAppliedConfiguration = lac
	}
	// kubectl.kubernetes.io/last-applied-configuration -> annotations
	lastAppliedAnnotationsPointer, _ := gojsonpointer.NewJsonPointer("/metadata/annotations")
	lastAppliedAnnotations, _, err := lastAppliedAnnotationsPointer.Get(i.LastAppliedConfiguration)
	if err == nil {
		i.LastAppliedAnnotations = lastAppliedAnnotations.(map[string]interface{})
	}

	// kubectl.kubernetes.io/last-applied-configuration -> container images
	// get all container image definitions, and paste them into the spec.
	if i.Kind == "DeploymentConfig" {
		containerSpecsPointer, _ := gojsonpointer.NewJsonPointer("/spec/template/spec/containers")
		appliedContainerSpecs, _, err := containerSpecsPointer.Get(i.LastAppliedConfiguration)
		if err == nil {
			for i, val := range appliedContainerSpecs.([]interface{}) {
				acs := val.(map[string]interface{})
				if appliedImageVal, ok := acs["image"]; ok {
					_, _, err := containerSpecsPointer.Get(m)
					if err == nil {
						imagePointer, _ := gojsonpointer.NewJsonPointer(fmt.Sprintf("/spec/template/spec/containers/%d/image", i))
						_, err := imagePointer.Set(m, appliedImageVal)
						if err != nil {
							cli.VerboseMsg("could not apply:", err.Error())
						}
					}
				}
			}
		} else { // backwards compatibility for pre 0.13.0
			tailorAppliedConfigAnnotation := "tailor.opendevstack.org/applied-config"
			escapedTailorAppliedConfigAnnotation := strings.Replace(tailorAppliedConfigAnnotation, "/", "~1", -1)
			tailorAppliedConfigAnnotationPath := annotationsPath + "/" + escapedTailorAppliedConfigAnnotation

			tailorAppliedConfigAnnotationPointer, err := gojsonpointer.NewJsonPointer(tailorAppliedConfigAnnotationPath)
			if err != nil {
				return fmt.Errorf("Could not create JSON pointer %s: %s", tailorAppliedConfigAnnotationPath, err)
			}
			val, _, err := tailorAppliedConfigAnnotationPointer.Get(m)
			if err == nil {
				valBytes := []byte(val.(string))
				tacFields := map[string]string{}
				err = json.Unmarshal(valBytes, &tacFields)
				if err != nil {
					return fmt.Errorf("Could not unmarshal JSON %s: %s", tailorAppliedConfigAnnotationPath, val)
				}
				for k, v := range tacFields {
					specPointer, err := gojsonpointer.NewJsonPointer(k)
					if err != nil {
						return fmt.Errorf("Could not create JSON pointer %s: %s", k, err)
					}
					_, err = specPointer.Set(m, v)
					if err != nil {
						return fmt.Errorf("Could not set %s: %s", k, err)
					}
				}
				_, err = tailorAppliedConfigAnnotationPointer.Delete(m)
				if err != nil {
					return fmt.Errorf("Could not delete %s: %s", tailorAppliedConfigAnnotationPath, err)
				}
			}
			delete(i.Annotations, tailorAppliedConfigAnnotation)
		}
	}

	// Remove platform-managed simple fields
	legacyFields := []string{"/userNames", "/groupNames"}
	for _, p := range platformManagedSimpleFields {
		deletePointer, _ := gojsonpointer.NewJsonPointer(p)
		_, _ = deletePointer.Delete(m)
		if utils.Includes(legacyFields, p) {
			cli.DebugMsg("Removed", p, "which is used for legacy clients, but not supported by Tailor")
		}
	}

	i.Config = m

	// Build list of JSON pointers
	i.walkMap(m, "")

	// Iterate over extracted paths and massage as necessary
	newPaths := []string{}
	deletedPathIndices := []int{}
	for pathIndex, path := range i.Paths {

		// Remove platform-managed regex fields
		for _, platformManagedField := range platformManagedRegexFields {
			matched, _ := regexp.MatchString(platformManagedField, path)
			if matched {
				deletePointer, _ := gojsonpointer.NewJsonPointer(path)
				_, _ = deletePointer.Delete(i.Config)
				deletedPathIndices = append(deletedPathIndices, pathIndex)
			}
		}
	}

	// As we delete items from a slice, we need to adjust the pre-calculated
	// indices to delete (shift to left by one for each deletion).
	indexOffset := 0
	for _, pathIndex := range deletedPathIndices {
		deletionIndex := pathIndex + indexOffset
		cli.DebugMsg("Removing platform managed path", i.Paths[deletionIndex])
		i.Paths = append(i.Paths[:deletionIndex], i.Paths[deletionIndex+1:]...)
		indexOffset = indexOffset - 1
	}
	if len(newPaths) > 0 {
		i.Paths = append(i.Paths, newPaths...)
	}

	return nil
}

func (i *ResourceItem) isImmutableField(field string) bool {
	for _, key := range immutableFields[i.Kind] {
		if key == field {
			return true
		}
	}
	return false
}

func (i *ResourceItem) walkMap(m map[string]interface{}, pointer string) {
	for k, v := range m {
		i.handleKeyValue(k, v, pointer)
	}
}

func (i *ResourceItem) walkArray(a []interface{}, pointer string) {
	for k, v := range a {
		i.handleKeyValue(k, v, pointer)
	}
}

func (i *ResourceItem) handleKeyValue(k interface{}, v interface{}, pointer string) {

	strK := ""
	switch kv := k.(type) {
	case string:
		strK = kv
	case int:
		strK = strconv.Itoa(kv)
	}

	relativePointer := utils.JSONPointerPath(strK)
	absolutePointer := pointer + "/" + relativePointer
	i.Paths = append(i.Paths, absolutePointer)

	switch vv := v.(type) {
	case []interface{}:
		i.walkArray(vv, absolutePointer)
	case map[string]interface{}:
		i.walkMap(vv, absolutePointer)
	}
}

func (i *ResourceItem) removeAnnotion(annotation string) {
	path := "/metadata/annotations/" + utils.JSONPointerPath(annotation)
	deletePointer, _ := gojsonpointer.NewJsonPointer(path)
	_, err := deletePointer.Delete(i.Config)
	if err != nil {
		cli.DebugMsg(fmt.Sprintf("Could not remove annotation %s from item", annotation))
	}
}

// prepareForComparisonWithPlatformItem massages template item in such a way
// that it can be compared with the given platform item:
// - copy value from platformItem to templateItem for externally modified paths
func (templateItem *ResourceItem) prepareForComparisonWithPlatformItem(platformItem *ResourceItem, preservePaths []string) error {
	for _, path := range preservePaths {
		cli.DebugMsg("Trying to preserve path", path, "in platform item", platformItem.FullName())
		pathPointer, _ := gojsonpointer.NewJsonPointer(path)
		platformItemVal, _, err := pathPointer.Get(platformItem.Config)
		if err != nil {
			cli.DebugMsg("No such path", path, "in platform item", platformItem.FullName())
			// As the current state for this path is "undefined" we need to make
			// sure that the desired state does not define any value for it,
			// otherwise it will show in the diff.
			_, _ = pathPointer.Delete(templateItem.Config)
		} else {
			_, err = pathPointer.Set(templateItem.Config, platformItemVal)
			if err != nil {
				cli.DebugMsg(fmt.Sprintf(
					"Could not set %s to %v in template item %s",
					path,
					platformItemVal,
					templateItem.FullName(),
				))
			} else {
				// Add preserved path and its subpaths to the paths slice
				// of the template item.
				templateItem.Paths = append(templateItem.Paths, path)
				switch vv := platformItemVal.(type) {
				case []interface{}:
					templateItem.walkArray(vv, path)
				case map[string]interface{}:
					templateItem.walkMap(vv, path)
				}
			}
		}
	}

	return nil
}

// prepareForComparisonWithTemplateItem massages platform item in such a way
// that it can be compared with the given template item:
// - remove all annotations which are not managed
func (platformItem *ResourceItem) prepareForComparisonWithTemplateItem(templateItem *ResourceItem) error {
	// Fix apiVersion
	// When running "oc process" on a template with a "Deployment" in
	// "apps/v1", and then running "oc export", the export contains
	// "apiVersion=extensions/v1beta1". If "oc process" is run *after*
	// "oc export", this issue is not present. Tailor runs "oc process" first
	// because it uncovers potential issues with local, desired state.
	// Therefore, we use the last applied apiVersion if we find
	// "apiVersion=extensions/v1beta1" so that no drift is reported.
	apiVersionPath := "/apiVersion"
	apiVersionPointer, _ := gojsonpointer.NewJsonPointer(apiVersionPath)
	apiVersion, _, err := apiVersionPointer.Get(platformItem.Config)
	if err == nil {
		lastAppliedAPIVersion, _, err := apiVersionPointer.Get(platformItem.LastAppliedConfiguration)
		if err == nil {
			if apiVersion.(string) == "extensions/v1beta1" {
				_, err := apiVersionPointer.Set(platformItem.Config, lastAppliedAPIVersion)
				if err != nil {
					cli.DebugMsg("could not set apiVersion:", err.Error())
				}
			}
		}
	}

	// Annotations
	unmanagedAnnotations := []string{}
	for a := range platformItem.Annotations {
		if _, ok := templateItem.Annotations[a]; ok {
			continue
		}
		if _, ok := platformItem.LastAppliedAnnotations[a]; ok {
			continue
		}
		unmanagedAnnotations = append(unmanagedAnnotations, a)
	}
	for _, a := range unmanagedAnnotations {
		path := "/metadata/annotations/" + utils.JSONPointerPath(a)
		deletePointer, _ := gojsonpointer.NewJsonPointer(path)
		_, err := deletePointer.Delete(platformItem.Config)
		if err != nil {
			return fmt.Errorf("Could not delete %s from configuration", path)
		}
		platformItem.Paths = utils.Remove(platformItem.Paths, path)
	}

	return nil
}
