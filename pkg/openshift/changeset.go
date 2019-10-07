package openshift

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/opendevstack/tailor/pkg/utils"
	"github.com/xeipuuv/gojsonpointer"
)

var (
	// Resources with no dependencies go first
	kindOrder = map[string]string{
		"Template":              "a",
		"ConfigMap":             "b",
		"Secret":                "c",
		"PersistentVolumeClaim": "d",
		"ImageStream":           "e",
		"BuildConfig":           "f",
		"DeploymentConfig":      "g",
		"Service":               "h",
		"Route":                 "i",
		"ServiceAccount":        "j",
		"RoleBinding":           "k",
	}
)

type Changeset struct {
	Create []*Change
	Update []*Change
	Delete []*Change
	Noop   []*Change
}

func NewChangeset(platformBasedList, templateBasedList *ResourceList, upsertOnly bool, allowRecreate bool, ignoredPaths []string) (*Changeset, error) {
	changeset := &Changeset{
		Create: []*Change{},
		Delete: []*Change{},
		Update: []*Change{},
		Noop:   []*Change{},
	}

	// items to delete
	if !upsertOnly {
		for _, item := range platformBasedList.Items {
			if _, err := templateBasedList.getItem(item.Kind, item.Name); err != nil {
				change := &Change{
					Action:       "Delete",
					Kind:         item.Kind,
					Name:         item.Name,
					CurrentState: item.YamlConfig(),
					DesiredState: "",
				}
				changeset.Add(change)
			}
		}
	}

	// items to create
	for _, item := range templateBasedList.Items {
		if _, err := platformBasedList.getItem(item.Kind, item.Name); err != nil {
			change := &Change{
				Action:       "Create",
				Kind:         item.Kind,
				Name:         item.Name,
				CurrentState: "",
				DesiredState: item.YamlConfig(),
			}
			changeset.Add(change)
		}
	}

	// items to update
	for _, templateItem := range templateBasedList.Items {
		platformItem, err := platformBasedList.getItem(
			templateItem.Kind,
			templateItem.Name,
		)
		if err == nil {
			externallyModifiedPaths := []string{}
			for _, path := range ignoredPaths {
				pathParts := strings.Split(path, ":")
				if len(pathParts) > 3 {
					return changeset, fmt.Errorf(
						"%s is not a valid ignore-path argument",
						path,
					)
				}
				// ignored paths can be either:
				// - globally (e.g. /spec/name)
				// - per-kind (e.g. bc:/spec/name)
				// - per-resource (e.g. bc:foo:/spec/name)
				if len(pathParts) == 1 ||
					(len(pathParts) == 2 &&
						templateItem.Kind == KindMapping[strings.ToLower(pathParts[0])]) ||
					(len(pathParts) == 3 &&
						templateItem.Kind == KindMapping[strings.ToLower(pathParts[0])] &&
						templateItem.Name == strings.ToLower(pathParts[1])) {
					// We only care about the last part (the JSON path) as we
					// are already "inside" the item
					externallyModifiedPaths = append(externallyModifiedPaths, pathParts[len(pathParts)-1])
				}
			}

			changes, err := calculateChanges(templateItem, platformItem, externallyModifiedPaths, allowRecreate)
			if err != nil {
				return changeset, err
			}
			changeset.Add(changes...)
		}
	}

	return changeset, nil
}

func calculateChanges(templateItem *ResourceItem, platformItem *ResourceItem, externallyModifiedPaths []string, allowRecreate bool) ([]*Change, error) {
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
				if allowRecreate {
					return recreateChanges(templateItem, platformItem), nil
				} else {
					return nil, fmt.Errorf("Path %s is immutable. Changing its value requires to delete and re-create the whole resource, which is only done when --allow-recreate is present", path)
				}

			}
			comparison[path] = &jsonPatch{Op: "add", Value: templateItemVal}
			addedPaths = append(addedPaths, path)
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
						if allowRecreate {
							return recreateChanges(templateItem, platformItem), nil
						} else {
							return nil, fmt.Errorf("Path %s is immutable. Changing its value requires to delete and re-create the whole resource, which is only done when --allow-recreate is present", path)
						}
					}
					comparison[path] = &jsonPatch{Op: "replace", Value: templateItemVal}
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

			pp, _ := gojsonpointer.NewJsonPointer(path)
			val, _, err := pp.Get(platformItem.Config)
			if err != nil {
				return nil, err
			}
			if val == nil {
				continue
			}

			// Skip annotations
			if path == annotationsPath {
				if x, ok := val.(map[string]interface{}); ok {
					if len(x) == 0 {
						pp.Set(templateItem.Config, map[string]interface{}{})
					}
				}
				continue
			}

			// If the value is an "empty value", there is no need to detect
			// drift for it. This allows template authors to reduce boilerplate
			// by omitting fields that have an "empty value".
			if x, ok := val.(map[string]interface{}); ok {
				if len(x) == 0 {
					pp.Set(templateItem.Config, map[string]interface{}{})
					continue
				}
			}
			if x, ok := val.([]interface{}); ok {
				if len(x) == 0 {
					pp.Set(templateItem.Config, []interface{}{})
					continue
				}
			}
			if x, ok := val.([]string); ok {
				if len(x) == 0 {
					pp.Set(templateItem.Config, []string{})
					continue
				}
			}

			// Pointer exist only in platformItem
			comparison[path] = &jsonPatch{Op: "remove"}
			deletedPaths = append(deletedPaths, path)
		}
	}

	// /metadata/annotations can be in 3 states: "add" (complex object),
	// "remove" or "noop". "replace" is NOT possible.

	// If it is "add", we might need to merge the hidden annotations into it.
	// If it is "remove", we might need to NOT remove it, but just add/replace
	// the hidden annotations.
	// If it is "noop", we do not need to care and just add/replace/remove the
	// hidden annotations.
	annotationsOp := "noop"
	annotationsMap := map[string]string{}
	if v, ok := comparison[annotationsPath]; ok {
		annotationsOp = v.Op
		if m, ok := v.Value.(map[string]interface{}); ok {
			for k, v := range m {
				annotationsMap[k] = v.(string)
			}
		}
	}

	// Add hidden JSON patches - we do not want to show those in the textual
	// diff to avoid unnecessary confusion for the enduser.
	// Managed annotations
	platformManagedAnnotations := strings.Join(platformItem.TailorManagedAnnotations, ",")
	templateManagedAnnotations := strings.Join(templateItem.TailorManagedAnnotations, ",")
	managedAnnotationsPatch := &jsonPatch{Op: "noop"}
	if platformManagedAnnotations != templateManagedAnnotations {
		if len(templateItem.TailorManagedAnnotations) == 0 {
			managedAnnotationsPatch = &jsonPatch{Op: "remove"}
		} else {
			managedAnnotationOp := "add"
			if len(platformItem.TailorManagedAnnotations) > 0 {
				managedAnnotationOp = "replace"
			}
			managedAnnotationsPatch = &jsonPatch{
				Op:    managedAnnotationOp,
				Value: templateManagedAnnotations,
			}
		}
	}
	if managedAnnotationsPatch.Op != "noop" {
		if annotationsOp == "add" {
			annotationsMap[escapedTailorManagedAnnotation] = managedAnnotationsPatch.Value.(string)
		} else {
			if annotationsOp == "remove" {
				comparison[annotationsPath].Op = "noop"
			}
			if platformItem.AnnotationsPresent {
				comparison[tailorManagedAnnotationPath] = managedAnnotationsPatch
			} else {
				comparison[annotationsPath] = &jsonPatch{Op: "add"}
				annotationsMap[escapedTailorManagedAnnotation] = managedAnnotationsPatch.Value.(string)
			}
		}
	}

	// Applied configuration
	appliedConfigAnnotationsPatch := &jsonPatch{Op: "noop"}
	if !reflect.DeepEqual(platformItem.TailorAppliedConfigFields, templateItem.TailorAppliedConfigFields) {
		if len(templateItem.TailorAppliedConfigFields) == 0 {
			appliedConfigAnnotationsPatch = &jsonPatch{Op: "remove"}
		} else {
			templateAppliedConfigAnnotation, err := json.Marshal(templateItem.TailorAppliedConfigFields)
			if err != nil {
				return nil, err
			}
			appliedConfigAnnotationOp := "add"
			if len(platformItem.TailorAppliedConfigFields) > 0 {
				appliedConfigAnnotationOp = "replace"
			}
			appliedConfigAnnotationsPatch = &jsonPatch{
				Op:    appliedConfigAnnotationOp,
				Value: string(templateAppliedConfigAnnotation),
			}
		}
	}
	if appliedConfigAnnotationsPatch.Op != "noop" {
		if annotationsOp == "add" {
			annotationsMap[escapedTailorAppliedConfigAnnotation] = appliedConfigAnnotationsPatch.Value.(string)
		} else {
			if annotationsOp == "remove" {
				comparison[annotationsPath].Op = "noop"
			}
			if platformItem.AnnotationsPresent {
				comparison[tailorAppliedConfigAnnotationPath] = appliedConfigAnnotationsPatch
			} else {
				comparison[annotationsPath] = &jsonPatch{Op: "add"}
				annotationsMap[escapedTailorAppliedConfigAnnotation] = appliedConfigAnnotationsPatch.Value.(string)
			}
		}
	}

	// Check if we need to set annotations as a complex key
	if v, ok := comparison[annotationsPath]; ok {
		v.Value = annotationsMap
	}

	c := NewChange(templateItem, platformItem, comparison)

	return []*Change{c}, nil
}

func (c *Changeset) Blank() bool {
	return len(c.Create) == 0 && len(c.Update) == 0 && len(c.Delete) == 0
}

func (c *Changeset) Add(changes ...*Change) {
	for _, change := range changes {
		switch change.Action {
		case "Create":
			c.Create = append(c.Create, change)
			sort.Slice(c.Create, func(i, j int) bool {
				return kindOrder[c.Create[i].Kind] < kindOrder[c.Create[j].Kind]
			})
		case "Update":
			c.Update = append(c.Update, change)
			sort.Slice(c.Update, func(i, j int) bool {
				return kindOrder[c.Update[i].Kind] < kindOrder[c.Update[j].Kind]
			})
		case "Delete":
			c.Delete = append(c.Delete, change)
			sort.Slice(c.Delete, func(i, j int) bool {
				return kindOrder[c.Delete[i].Kind] > kindOrder[c.Delete[j].Kind]
			})
		case "Noop":
			c.Noop = append(c.Noop, change)
		}
	}
}
