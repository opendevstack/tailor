package openshift

var (
	kindToShortMapping = map[string]string{
		"Service":               "svc",
		"Route":                 "route",
		"DeploymentConfig":      "dc",
		"BuildConfig":           "bc",
		"ImageStream":           "is",
		"PersistentVolumeClaim": "pvc",
		"Template":              "template",
		"ConfigMap":             "cm",
		"Secret":                "secret",
		"RoleBinding":           "rolebinding",
		"ServiceAccount":        "serviceaccount",
	}
)

type Changeset struct {
	Create []*Change
	Update []*Change
	Delete []*Change
	Noop   []*Change
}

type Change struct {
	Kind         string
	Name         string
	CurrentState string
	DesiredState string
}

func NewChangeset(remoteResourceList, localResourceList *ResourceList, upsertOnly bool) *Changeset {
	changeset := &Changeset{
		Create: []*Change{},
		Delete: []*Change{},
		Update: []*Change{},
		Noop:   []*Change{},
	}

	// items to delete
	if !upsertOnly {
		for _, item := range remoteResourceList.Items {
			if _, err := localResourceList.GetItem(item.Kind, item.Name); err != nil {
				change := &Change{
					Kind:         item.Kind,
					Name:         item.Name,
					CurrentState: item.YamlConfig(),
					DesiredState: "",
				}
				changeset.Delete = append(changeset.Delete, change)
			}
		}
	}

	// items to create
	for _, item := range localResourceList.Items {
		if _, err := remoteResourceList.GetItem(item.Kind, item.Name); err != nil {
			change := &Change{
				Kind:         item.Kind,
				Name:         item.Name,
				CurrentState: "",
				DesiredState: item.YamlConfig(),
			}
			changeset.Create = append(changeset.Create, change)
		}
	}

	// items to update
	for _, lItem := range localResourceList.Items {
		rItem, err := remoteResourceList.GetItem(lItem.Kind, lItem.Name)
		if err == nil {
			currentItemConfig := rItem.YamlConfig()
			desiredItemConfig := lItem.YamlConfig()
			change := &Change{
				Kind:         lItem.Kind,
				Name:         lItem.Name,
				CurrentState: currentItemConfig,
				DesiredState: desiredItemConfig,
			}
			if currentItemConfig == desiredItemConfig {
				changeset.Noop = append(changeset.Noop, change)
			} else if lItem.ImmutableFieldsEqual(rItem) {
				changeset.Update = append(changeset.Update, change)
			} else {
				deleteChange := &Change{
					Kind:         lItem.Kind,
					Name:         lItem.Name,
					CurrentState: currentItemConfig,
					DesiredState: "",
				}
				changeset.Delete = append(changeset.Delete, deleteChange)
				createChange := &Change{
					Kind:         lItem.Kind,
					Name:         lItem.Name,
					CurrentState: "",
					DesiredState: desiredItemConfig,
				}
				changeset.Create = append(changeset.Create, createChange)
			}
		}
	}

	return changeset
}

func (c *Change) ItemName() string {
	return kindToShortMapping[c.Kind] + "/" + c.Name
}

func (c *Changeset) Blank() bool {
	return len(c.Create) == 0 && len(c.Update) == 0 && len(c.Delete) == 0
}
