package openshift

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonpointer"
)

var (
	tailorManagedAnnotation = "managed-annotations.tailor.opendevstack.org"
	//tailorIgnoredAnnotation = "ignored-fields.tailor.opendevstack.org"
	platformManagedFields = []string{
		"/metadata/generation",
		"/metadata/creationTimestamp",
		"/spec/tags",
		"/status",
		"/spec/volumeName",
		"/spec/template/metadata/creationTimestamp",
	}
	emptyMapFields = []string{
		"/metadata/annotations",
	}
	immutableFields = map[string][]string{
		"Route": []string{
			"/spec/host",
		},
		"PersistentVolumeClaim": []string{
			"/spec/accessModes",
			"/spec/storageClassName",
			"/spec/resources/requests/storage",
		},
	}
	platformModifiedFields = []string{
		"/spec/template/spec/containers/[0-9]+/image$",
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
	//TailorIgnoredFields      []string
}

func NewResourceItem(m map[string]interface{}, source string) (*ResourceItem, error) {
	item := &ResourceItem{Source: source}
	err := item.ParseConfig(m)
	return item, err
}

func (i *ResourceItem) FullName() string {
	return i.Kind + "/" + i.Name
}

func (templateItem *ResourceItem) ChangesFrom(platformItem *ResourceItem) []*Change {
	comparison := map[string]*JsonPatch{}
	addedPaths := []string{}

	for _, path := range templateItem.Paths {
		// // Skip ignored fields
		// for _, i := range templateItem.TailorIgnoredFields {
		// 	if path == i {
		// 		// TODO: Delete path from platformItem
		// 		continue
		// 	}
		// }

		// Skip subpaths of already added paths
		skip := false
		for _, addedPath := range addedPaths {
			if strings.HasPrefix(path, addedPath) {
				skip = true
			}
		}
		if skip {
			continue
		}

		pathPointer, _ := gojsonpointer.NewJsonPointer(path)
		templateItemVal, _, _ := pathPointer.Get(templateItem.Config)
		platformItemVal, _, err := pathPointer.Get(platformItem.Config)

		if err != nil {
			// Pointer does not exist in platformItem
			if templateItem.isImmutableField(path) {
				return recreateChanges(templateItem, platformItem)
			} else {
				comparison[path] = &JsonPatch{Op: "add", Value: templateItemVal}
				addedPaths = append(addedPaths, path)
			}
		} else {
			// Pointer exists in both items
			switch templateItemVal.(type) {
			case []interface{}:
				// slice content changed, continue ...
				comparison[path] = &JsonPatch{Op: "noop"}
			case []string:
				// slice content changed, continue ...
				comparison[path] = &JsonPatch{Op: "noop"}
			case map[string]interface{}:
				// map content changed, continue
				comparison[path] = &JsonPatch{Op: "noop"}
			default:
				if templateItemVal == platformItemVal {
					comparison[path] = &JsonPatch{Op: "noop"}
				} else {
					if templateItem.isImmutableField(path) {
						return recreateChanges(templateItem, platformItem)
					} else {
						comparison[path] = &JsonPatch{Op: "replace", Value: templateItemVal}
					}
				}
			}
		}
	}

	deletedPaths := []string{}

	for _, path := range platformItem.Paths {
		// Skip ignored fields
		// for _, i := range templateItem.TailorIgnoredFields {
		// 	if path == i {
		// 		// TODO: Delete path from platformItem
		// 		continue
		// 	}
		// }
		if _, ok := comparison[path]; !ok {
			// If path is an annotation and is currently managed
			// by Tailor on the platform, we do not want to delete it.
			if strings.HasPrefix(path, "/metadata/annotations") && path != "/metadata/annotations" {
				tailorManaged := false
				a := strings.Replace(path, "/metadata/annotations/", "", -1)
				for _, v := range platformItem.TailorManagedAnnotations {
					if a == v {
						tailorManaged = true
						break
					}
				}
				if !tailorManaged && a != tailorManagedAnnotation {
					deletePointer, _ := gojsonpointer.NewJsonPointer(path)
					_, _ = deletePointer.Delete(platformItem.Config)
					continue
				}
			}

			// Do not delete subpaths of already deleted paths
			skip := false
			for _, deletedPath := range deletedPaths {
				if strings.HasPrefix(path, deletedPath) {
					skip = true
				}
			}
			if skip {
				continue
			}

			// Pointer exist only in platformItem
			comparison[path] = &JsonPatch{Op: "remove"}
			deletedPaths = append(deletedPaths, path)
		}
	}

	c := &Change{
		Kind:         templateItem.Kind,
		Name:         templateItem.Name,
		Patches:      []*JsonPatch{},
		CurrentState: platformItem.YamlConfig(),
		DesiredState: templateItem.YamlConfig(),
	}

	for path, patch := range comparison {
		if patch.Op != "noop" {
			patch.Path = path
			c.AddPatch(patch)
		}
	}

	if len(c.Patches) > 0 {
		c.Action = "Update"
	} else {
		c.Action = "Noop"
	}

	return []*Change{c}
}

func (i *ResourceItem) YamlConfig() string {
	y, _ := yaml.Marshal(i.Config)
	return string(y)
}

func (i *ResourceItem) ParseConfig(m map[string]interface{}) error {
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
	// If some annotations are managed, the ones that are *not* managed can be
	// removed - they are effectively platform-managed fields.
	i.TailorManagedAnnotations = []string{}
	if i.Source == "platform" {
		p, _ := gojsonpointer.NewJsonPointer("/metadata/annotations/" + tailorManagedAnnotation)
		managedAnnotations, _, err := p.Get(m)
		if err == nil {
			i.TailorManagedAnnotations = strings.Split(managedAnnotations.(string), ",")
		}
	} else { // source = template
		for k, _ := range i.Annotations {
			i.TailorManagedAnnotations = append(i.TailorManagedAnnotations, k)
		}
		if len(i.TailorManagedAnnotations) > 0 {
			p, _ := gojsonpointer.NewJsonPointer("/metadata/annotations/" + tailorManagedAnnotation)
			sort.Strings(i.TailorManagedAnnotations)
			p.Set(m, strings.Join(i.TailorManagedAnnotations, ","))
		}
	}

	// Remove platform-managed fields
	for _, p := range platformManagedFields {
		deletePointer, _ := gojsonpointer.NewJsonPointer(p)
		_, _ = deletePointer.Delete(m)
	}

	// Ignored fields
	// i.TailorIgnoredFields = []string{}
	// p, _ := gojsonpointer.NewJsonPointer("/metadata/annotations/" + tailorIgnoredAnnotation)
	// ignoredAnnotation, _, err := p.Get(m)
	// if err == nil {
	// 	i.TailorIgnoredFields = strings.Split(ignoredAnnotation.(string), ",")
	// }

	i.Config = m

	// Build list of JSON pointers
	i.walkMap(m, "")

	// Handle platform-modified fields:
	// If there is an annotation, copy its value into the spec, otherwise
	// copy the spec value into the annotation.
	for _, path := range i.Paths {
		for _, platformModifiedField := range platformModifiedFields {
			matched, _ := regexp.MatchString(platformModifiedField, path)
			if matched {
				annotationKey := strings.Replace(strings.TrimLeft(path, "/"), "/", ".", -1)
				annotationPath := "/metadata/annotations/original-values.tailor.io~1" + annotationKey
				annotationPointer, _ := gojsonpointer.NewJsonPointer(annotationPath)
				specPointer, _ := gojsonpointer.NewJsonPointer(path)
				specValue, _, _ := specPointer.Get(i.Config)
				annotationValue, _, err := annotationPointer.Get(i.Config)
				if err == nil {
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
					}
					_, err = annotationPointer.Set(i.Config, specValue)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
func (i *ResourceItem) RemoveUnmanagedAnnotations() {
	for a, _ := range i.Annotations {
		managed := false
		for _, m := range i.TailorManagedAnnotations {
			if a == m {
				managed = true
			}
		}
		if !managed {
			deletePointer, _ := gojsonpointer.NewJsonPointer("/metadata/annotations/" + a)
			deletePointer.Delete(i.Config)
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

	// Build JSON pointer according to spec, see
	// https://tools.ietf.org/html/draft-ietf-appsawg-json-pointer-07#section-3.
	relativePointer := strings.Replace(strK, "~", "~0", -1)
	relativePointer = strings.Replace(relativePointer, "/", "~1", -1)
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
