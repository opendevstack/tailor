package openshift

import (
	"github.com/opendevstack/tailor/cli"
)

type Changeset struct {
	Create []*Change
	Update []*Change
	Delete []*Change
	Noop   []*Change
}

func NewChangeset(platformBasedList, templateBasedList *ResourceList, upsertOnly bool) *Changeset {
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
			changes := templateItem.ChangesFrom(platformItem)
			changeset.Add(changes...)
		}
	}

	return changeset
}

func (c *Changeset) Blank() bool {
	return len(c.Create) == 0 && len(c.Update) == 0 && len(c.Delete) == 0
}

func (c *Changeset) Add(changes ...*Change) {
	for _, change := range changes {
		switch change.Action {
		case "Create":
			c.Create = append(c.Create, change)
		case "Update":
			c.Update = append(c.Update, change)
		case "Delete":
			c.Delete = append(c.Delete, change)
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
