package openshift

import (
	"github.com/pmezard/go-difflib/difflib"
)

var (
	kindToShortMapping = map[string]string{
		"Service":               "svc",
		"Route":                 "route",
		"DeploymentConfig":      "dc",
		"Deployment":            "deployment",
		"BuildConfig":           "bc",
		"ImageStream":           "is",
		"PersistentVolumeClaim": "pvc",
		"Template":              "template",
		"ConfigMap":             "cm",
		"Secret":                "secret",
		"RoleBinding":           "rolebinding",
		"ServiceAccount":        "serviceaccount",
		"CronJob":               "cronjob",
		"LimitRange":            "limitrange",
		"ResourceQuota":         "quota",
	}
)

// Change is a description of a drift between current and desired state, and
// the required patches to bring them back in sync.
type Change struct {
	Action       string
	Kind         string
	Name         string
	CurrentState string
	DesiredState string
}

// NewChange creates a new change for given template/platform item.
func NewChange(templateItem *ResourceItem, platformItem *ResourceItem) *Change {
	c := &Change{
		Kind:         templateItem.Kind,
		Name:         templateItem.Name,
		CurrentState: platformItem.YamlConfig(),
		DesiredState: templateItem.YamlConfig(),
	}

	if platformItem.YamlConfig() != templateItem.YamlConfig() {
		c.Action = "Update"
	} else {
		c.Action = "Noop"
	}

	return c
}

// ItemName returns the kind/name of the resource the change relates to.
func (c *Change) ItemName() string {
	return kindToShortMapping[c.Kind] + "/" + c.Name
}

// Diff returns a unified diff text for the change.
func (c *Change) Diff(revealSecrets bool) string {
	if c.isSecret() && !revealSecrets {
		return "Secret drift is hidden. Use --reveal-secrets to see details.\n"
	}
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(c.CurrentState),
		B:        difflib.SplitLines(c.DesiredState),
		FromFile: "Current State (OpenShift cluster)",
		ToFile:   "Desired State (Processed template)",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

func (c *Change) isSecret() bool {
	return kindToShortMapping[c.Kind] == "secret"
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
