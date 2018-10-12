package openshift

import (
	"fmt"
	"sort"
	"strings"

	"github.com/opendevstack/tailor/cli"
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

func NewChangeset(platformBasedList, templateBasedList *ResourceList, upsertOnly bool, ignoredPaths []string) (*Changeset, error) {
	changeset := &Changeset{
		Create: []*Change{},
		Delete: []*Change{},
		Update: []*Change{},
		Noop:   []*Change{},
	}

	// items to delete
	if !upsertOnly {
		for _, item := range platformBasedList.Items {
			if _, err := templateBasedList.GetItem(item.Kind, item.Name); err != nil {
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
		if _, err := platformBasedList.GetItem(item.Kind, item.Name); err != nil {
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
		platformItem, err := platformBasedList.GetItem(
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
				if len(pathParts) == 1 ||
					(len(pathParts) == 2 &&
						templateItem.Kind == KindMapping[strings.ToLower(pathParts[0])]) ||
					(len(pathParts) == 3 &&
						templateItem.Kind == KindMapping[strings.ToLower(pathParts[0])] &&
						templateItem.Name == strings.ToLower(pathParts[1])) {
					externallyModifiedPaths = append(externallyModifiedPaths, pathParts[len(pathParts)-1])
				}
			}

			changes, err := templateItem.ChangesFrom(platformItem, externallyModifiedPaths)
			if err != nil {
				return changeset, err
			}
			changeset.Add(changes...)
		}
	}

	return changeset, nil
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

func (c *Changeset) Apply(compareOptions *cli.CompareOptions) error {
	for _, change := range c.Create {
		err := ocCreate(change, compareOptions)
		if err != nil {
			return err
		}
	}

	for _, change := range c.Delete {
		err := ocDelete(change, compareOptions)
		if err != nil {
			return err
		}
	}

	for _, change := range c.Update {
		err := ocPatch(change, compareOptions)
		if err != nil {
			return err
		}
	}

	return nil
}
