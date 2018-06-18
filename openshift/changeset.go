package openshift

type Changeset struct {
	Create map[string][]string
	Update map[string][]string
	Delete map[string][]string
	Noop   map[string][]string
}

func (c *Changeset) Blank() bool {
	return len(c.Create) == 0 && len(c.Update) == 0 && len(c.Delete) == 0
}

func NewChangeset(remoteResourceList, localResourceList *ResourceList) *Changeset {
	changeset := &Changeset{
		Create: make(map[string][]string),
		Delete: make(map[string][]string),
		Update: make(map[string][]string),
		Noop:   make(map[string][]string),
	}

	// items to delete
	for _, item := range remoteResourceList.Items {
		if _, err := localResourceList.GetItem(item.Name); err != nil {
			changeset.Delete[item.Name] = []string{item.YamlConfig(), ""}
		}
	}

	// items to create
	for _, item := range localResourceList.Items {
		if _, err := remoteResourceList.GetItem(item.Name); err != nil {
			changeset.Create[item.Name] = []string{"", item.YamlConfig()}
		}
	}

	// items to update
	for _, lItem := range localResourceList.Items {
		rItem, err := remoteResourceList.GetItem(lItem.Name)
		if err == nil {
			currentItemConfig := rItem.YamlConfig()
			desiredItemConfig := lItem.YamlConfig()
			if currentItemConfig == desiredItemConfig {
				changeset.Noop[lItem.Name] = []string{currentItemConfig, desiredItemConfig}
			} else {
				changeset.Update[lItem.Name] = []string{currentItemConfig, desiredItemConfig}
			}
		}
	}

	return changeset
}
