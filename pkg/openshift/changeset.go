package openshift

import (
	"fmt"
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
		"LimitRange":            "d",
		"ResourceQuota":         "e",
		"PersistentVolumeClaim": "f",
		"CronJob":               "g",
		"ImageStream":           "h",
		"BuildConfig":           "i",
		"DeploymentConfig":      "j",
		"Deployment":            "k",
		"Service":               "l",
		"Route":                 "m",
		"ServiceAccount":        "n",
		"RoleBinding":           "o",
	}
)

type Changeset struct {
	Create []*Change
	Update []*Change
	Delete []*Change
	Noop   []*Change
}

func NewChangeset(platformBasedList, templateBasedList *ResourceList, upsertOnly bool, allowRecreate bool, preservePaths []string) (*Changeset, error) {
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
			desiredState, err := item.DesiredConfig()
			if err != nil {
				return changeset, err
			}
			change := &Change{
				Action:       "Create",
				Kind:         item.Kind,
				Name:         item.Name,
				CurrentState: "",
				DesiredState: desiredState,
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
			actualReservePaths := []string{}
			for _, path := range preservePaths {
				pathParts := strings.Split(path, ":")
				if len(pathParts) > 3 {
					return changeset, fmt.Errorf(
						"%s is not a valid preserve argument",
						path,
					)
				}
				// Preserved paths can be either:
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
					actualReservePaths = append(actualReservePaths, pathParts[len(pathParts)-1])
				}
			}

			changes, err := calculateChanges(templateItem, platformItem, actualReservePaths, allowRecreate)
			if err != nil {
				return changeset, err
			}
			changeset.Add(changes...)
		}
	}

	return changeset, nil
}

func calculateChanges(templateItem *ResourceItem, platformItem *ResourceItem, preservePaths []string, allowRecreate bool) ([]*Change, error) {
	err := templateItem.prepareForComparisonWithPlatformItem(platformItem, preservePaths)
	if err != nil {
		return nil, err
	}
	err = platformItem.prepareForComparisonWithTemplateItem(templateItem)
	if err != nil {
		return nil, err
	}

	comparedPaths := map[string]bool{}
	addedPaths := []string{}

	for _, path := range templateItem.Paths {

		// Skip subpaths of already added paths
		if utils.IncludesPrefix(addedPaths, path) {
			continue
		}

		// Paths that should be preserved are no-ops
		if utils.IncludesPrefix(preservePaths, path) {
			comparedPaths[path] = true
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
			comparedPaths[path] = true

			// OpenShift sometimes removes the whole field when the value is an
			// empty string. Therefore, we do not want to add the path in that
			// case, otherwise we would cause endless drift. See
			// https://github.com/opendevstack/tailor/issues/157.
			if v, ok := templateItemVal.(string); ok && len(v) == 0 {
				_, err := pathPointer.Delete(templateItem.Config)
				if err != nil {
					return nil, err
				}
			} else {
				addedPaths = append(addedPaths, path)
			}
		} else {
			// Pointer exists in both items
			switch templateItemVal.(type) {
			case []interface{}:
				// slice content changed, continue ...
				comparedPaths[path] = true
			case []string:
				// slice content changed, continue ...
				comparedPaths[path] = true
			case map[string]interface{}:
				// map content changed, continue
				comparedPaths[path] = true
			default:
				if templateItemVal == platformItemVal {
					comparedPaths[path] = true
				} else {
					if templateItem.isImmutableField(path) {
						if allowRecreate {
							return recreateChanges(templateItem, platformItem), nil
						} else {
							return nil, fmt.Errorf("Path %s is immutable. Changing its value requires to delete and re-create the whole resource, which is only done when --allow-recreate is present", path)
						}
					}
					comparedPaths[path] = true
				}
			}
		}
	}

	deletedPaths := []string{}

	for _, path := range platformItem.Paths {
		if _, ok := comparedPaths[path]; !ok {
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
						_, err := pp.Set(templateItem.Config, map[string]interface{}{})
						if err != nil {
							return nil, err
						}
					}
				}
				continue
			}

			// If the value is an "empty value", there is no need to detect
			// drift for it. This allows template authors to reduce boilerplate
			// by omitting fields that have an "empty value".
			if x, ok := val.(map[string]interface{}); ok {
				if len(x) == 0 {
					_, err := pp.Set(templateItem.Config, map[string]interface{}{})
					if err != nil {
						return nil, err
					}
					continue
				}
			}
			if x, ok := val.([]interface{}); ok {
				if len(x) == 0 {
					_, err := pp.Set(templateItem.Config, []interface{}{})
					if err != nil {
						return nil, err
					}
					continue
				}
			}
			if x, ok := val.([]string); ok {
				if len(x) == 0 {
					_, err := pp.Set(templateItem.Config, []string{})
					if err != nil {
						return nil, err
					}
					continue
				}
			}

			// Pointer exist only in platformItem
			comparedPaths[path] = true
			deletedPaths = append(deletedPaths, path)
		}
	}

	c := NewChange(templateItem, platformItem)

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
