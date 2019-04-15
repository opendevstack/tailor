package openshift

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/utils"
	"github.com/xeipuuv/gojsonpointer"
)

var (
	tailorOriginalValuesAnnotationPrefix = "original-values.tailor.io"
	tailorManagedAnnotation              = "managed-annotations.tailor.opendevstack.org"
	platformManagedSimpleFields          = []string{
		"/metadata/generation",
		"/metadata/creationTimestamp",
		"/spec/tags",
		"/status",
		"/spec/volumeName",
		"/spec/template/metadata/creationTimestamp",
	}
	platformManagedRegexFields = []string{
		"^/spec/triggers/[0-9]*/imageChangeParams/lastTriggeredImage",
	}
	emptyMapFields = []string{
		"/metadata/annotations",
		"/spec/template/metadata/annotations",
	}
	immutableFields = map[string][]string{
		"PersistentVolumeClaim": []string{
			"/spec/accessModes",
			"/spec/storageClassName",
			"/spec/resources/requests/storage",
		},
		"Route": []string{
			"/spec/host",
		},
		"Secret": []string{
			"/type",
		},
	}
	platformModifiedFields = []string{
		"/spec/template/spec/containers/[0-9]+/image$",
	}

	KindMapping = map[string]string{
		"svc":                   "Service",
		"service":               "Service",
		"route":                 "Route",
		"dc":                    "DeploymentConfig",
		"deploymentconfig":      "DeploymentConfig",
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
	TailorManagedAnnotations []string
}

func NewResourceItem(m map[string]interface{}, source string) (*ResourceItem, error) {
	item := &ResourceItem{Source: source}
	err := item.parseConfig(m)
	return item, err
}

func (i *ResourceItem) FullName() string {
	return i.Kind + "/" + i.Name
}

func (templateItem *ResourceItem) ChangesFrom(platformItem *ResourceItem, externallyModifiedPaths []string) ([]*Change, error) {
	err := templateItem.prepareForComparisonWithPlatformItem(platformItem, externallyModifiedPaths)
	if err != nil {
		return nil, err
	}
	err = platformItem.prepareForComparisonWithTemplateItem(templateItem)
	if err != nil {
		return nil, err
	}

	comparison := map[string]*jsonPatch{}
	addedPaths := []string{}

	for _, path := range templateItem.Paths {
		// Skip subpaths of already added paths
		if utils.IncludesPrefix(addedPaths, path) {
			continue
		}

		pathPointer, _ := gojsonpointer.NewJsonPointer(path)
		templateItemVal, _, _ := pathPointer.Get(templateItem.Config)
		platformItemVal, _, err := pathPointer.Get(platformItem.Config)

		if err != nil {
			// Pointer does not exist in platformItem
			if templateItem.isImmutableField(path) {
				return recreateChanges(templateItem, platformItem), nil
			} else {
				comparison[path] = &jsonPatch{Op: "add", Value: templateItemVal}
				addedPaths = append(addedPaths, path)
			}
		} else {
			// Pointer exists in both items
			switch templateItemVal.(type) {
			case []interface{}:
				// slice content changed, continue ...
				comparison[path] = &jsonPatch{Op: "noop"}
			case []string:
				// slice content changed, continue ...
				comparison[path] = &jsonPatch{Op: "noop"}
			case map[string]interface{}:
				// map content changed, continue
				comparison[path] = &jsonPatch{Op: "noop"}
			default:
				if templateItemVal == platformItemVal {
					comparison[path] = &jsonPatch{Op: "noop"}
				} else {
					if templateItem.isImmutableField(path) {
						return recreateChanges(templateItem, platformItem), nil
					} else {
						comparison[path] = &jsonPatch{Op: "replace", Value: templateItemVal}
					}
				}
			}
		}
	}

	deletedPaths := []string{}

	for _, path := range platformItem.Paths {
		if _, ok := comparison[path]; !ok {
			// Do not delete subpaths of already deleted paths
			if utils.IncludesPrefix(deletedPaths, path) {
				continue
			}
			// Pointer exist only in platformItem
			comparison[path] = &jsonPatch{Op: "remove"}
			deletedPaths = append(deletedPaths, path)
		}
	}

	c := &Change{
		Kind:         templateItem.Kind,
		Name:         templateItem.Name,
		Patches:      []*jsonPatch{},
		CurrentState: platformItem.YamlConfig(),
		DesiredState: templateItem.YamlConfig(),
	}

	for path, patch := range comparison {
		if patch.Op != "noop" {
			cli.DebugMsg("add path", path)
			patch.Path = path
			c.addPatch(patch)
		}
	}

	if len(c.Patches) > 0 {
		c.Action = "Update"
	} else {
		c.Action = "Noop"
	}

	return []*Change{c}, nil
}

func (i *ResourceItem) YamlConfig() string {
	y, _ := yaml.Marshal(i.Config)
	return string(y)
}

// parseConfig uses the config to initialise an item. The logic is the same
// for template and platform items, with no knowledge of the "other" item - it
// may or may not exist.
func (i *ResourceItem) parseConfig(m map[string]interface{}) error {
	// Extract kind and name
	kindPointer, _ := gojsonpointer.NewJsonPointer("/kind")
	kind, _, err := kindPointer.Get(m)
	if err != nil {
		return err
	}
	i.Kind = kind.(string)
	namePointer, _ := gojsonpointer.NewJsonPointer("/metadata/name")
	name, _, err := namePointer.Get(m)
	if err != nil {
		return err
	}
	i.Name = name.(string)

	// Extract labels
	labelsPointer, _ := gojsonpointer.NewJsonPointer("/metadata/labels")
	labels, _, err := labelsPointer.Get(m)
	if err != nil {
		i.Labels = make(map[string]interface{})
	} else {
		i.Labels = labels.(map[string]interface{})
	}

	// Add empty maps
	for _, p := range emptyMapFields {
		initPointer, _ := gojsonpointer.NewJsonPointer(p)
		_, _, err := initPointer.Get(m)
		if err != nil {
			initPointer.Set(m, make(map[string]interface{}))
		}
	}

	// Extract annotations
	annotationsPointer, _ := gojsonpointer.NewJsonPointer("/metadata/annotations")
	annotations, _, err := annotationsPointer.Get(m)
	i.Annotations = make(map[string]interface{})
	if err == nil {
		for k, v := range annotations.(map[string]interface{}) {
			i.Annotations[k] = v
		}
	}

	// Figure out which annotations are managed by Tailor
	i.TailorManagedAnnotations = []string{}
	if i.Source == "platform" {
		// For platform items, only annotation listed in tailorManagedAnnotation are managed
		p, _ := gojsonpointer.NewJsonPointer("/metadata/annotations/" + tailorManagedAnnotation)
		managedAnnotations, _, err := p.Get(m)
		if err == nil {
			i.TailorManagedAnnotations = strings.Split(managedAnnotations.(string), ",")
		}
	} else { // source = template
		// For template items, all annotations are managed
		for k, _ := range i.Annotations {
			i.TailorManagedAnnotations = append(i.TailorManagedAnnotations, k)
		}
		// If there are any managed annotations, we need to set tailorManagedAnnotation
		if len(i.TailorManagedAnnotations) > 0 {
			p, _ := gojsonpointer.NewJsonPointer("/metadata/annotations/" + tailorManagedAnnotation)
			sort.Strings(i.TailorManagedAnnotations)
			p.Set(m, strings.Join(i.TailorManagedAnnotations, ","))
		}
	}

	// Remove platform-managed simple fields
	for _, p := range platformManagedSimpleFields {
		deletePointer, _ := gojsonpointer.NewJsonPointer(p)
		_, _ = deletePointer.Delete(m)
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

		// Deal with platform-modified fields
		// If there is an annotation, copy its value into the spec, otherwise
		// copy the spec value into the annotation.
		for _, platformModifiedField := range platformModifiedFields {
			matched, _ := regexp.MatchString(platformModifiedField, path)
			if matched {
				annotationKey := strings.Replace(strings.TrimLeft(path, "/"), "/", ".", -1)
				annotationPath := "/metadata/annotations/" + tailorOriginalValuesAnnotationPrefix + "~1" + annotationKey
				annotationPointer, _ := gojsonpointer.NewJsonPointer(annotationPath)
				specPointer, _ := gojsonpointer.NewJsonPointer(path)
				specValue, _, _ := specPointer.Get(i.Config)
				annotationValue, _, err := annotationPointer.Get(i.Config)
				if err == nil {
					cli.DebugMsg("Platform: Setting", path, "to", annotationValue.(string))
					_, err := specPointer.Set(i.Config, annotationValue)
					if err != nil {
						return err
					}
				} else {
					// Ensure there is an annotation map before setting values in it
					anP, _ := gojsonpointer.NewJsonPointer("/metadata/annotations")
					_, _, err := anP.Get(i.Config)
					if err != nil {
						anP.Set(i.Config, map[string]interface{}{})
						newPaths = append(newPaths, "/metadata/annotations")
					}
					cli.DebugMsg("Template: Setting", annotationPath, "to", specValue.(string))
					_, err = annotationPointer.Set(i.Config, specValue)
					if err != nil {
						return err
					}
					newPaths = append(newPaths, annotationPath)
				}
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
func (i *ResourceItem) RemoveUnmanagedAnnotations() {
	for a := range i.Annotations {
		managed := false
		for _, m := range i.TailorManagedAnnotations {
			if a == m {
				managed = true
			}
		}
		if !managed {
			cli.DebugMsg("Removing unmanaged annotation", a)
			path := "/metadata/annotations/" + utils.JSONPointerPath(a)
			deletePointer, _ := gojsonpointer.NewJsonPointer(path)
			_, err := deletePointer.Delete(i.Config)
			if err != nil {
				cli.DebugMsg("WARN: Could not remove unmanaged annotation", a)
				fmt.Printf("%v", i.Config)
			}
		}
	}
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

func recreateChanges(templateItem, platformItem *ResourceItem) []*Change {
	deleteChange := &Change{
		Action:       "Delete",
		Kind:         templateItem.Kind,
		Name:         templateItem.Name,
		CurrentState: platformItem.YamlConfig(),
		DesiredState: "",
	}
	createChange := &Change{
		Action:       "Create",
		Kind:         templateItem.Kind,
		Name:         templateItem.Name,
		CurrentState: "",
		DesiredState: templateItem.YamlConfig(),
	}
	return []*Change{deleteChange, createChange}
}

// prepareForComparisonWithPlatformItem massages template item in such a way
// that it can be compared with the given platform item:
// - copy value from platformItem to templateItem for externally modified paths
func (templateItem *ResourceItem) prepareForComparisonWithPlatformItem(platformItem *ResourceItem, externallyModifiedPaths []string) error {
	for _, path := range externallyModifiedPaths {
		pathPointer, _ := gojsonpointer.NewJsonPointer(path)
		platformItemVal, _, err := pathPointer.Get(platformItem.Config)
		if err != nil {
			cli.DebugMsg("No such path", path, "in platform item", platformItem.FullName())
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
				// Add ignored path and its subpaths to the paths slice
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
	unmanagedAnnotations := []string{}
	for a, _ := range platformItem.Annotations {
		if a == tailorManagedAnnotation {
			continue
		}
		if strings.HasPrefix(a, tailorOriginalValuesAnnotationPrefix) {
			continue
		}
		if utils.Includes(templateItem.TailorManagedAnnotations, a) {
			continue
		}
		if utils.Includes(platformItem.TailorManagedAnnotations, a) {
			continue
		}
		unmanagedAnnotations = append(unmanagedAnnotations, a)
	}
	for _, a := range unmanagedAnnotations {
		path := "/metadata/annotations/" + utils.JSONPointerPath(a)
		cli.DebugMsg("Deleting unmanaged annotation", path)
		deletePointer, _ := gojsonpointer.NewJsonPointer(path)
		_, err := deletePointer.Delete(platformItem.Config)
		if err != nil {
			return fmt.Errorf("Could not delete %s from configuration", path)
		}
		platformItem.Paths = utils.Remove(platformItem.Paths, path)
	}
	return nil
}
