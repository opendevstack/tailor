package openshift

import (
	"encoding/json"
	"sort"

	"github.com/pmezard/go-difflib/difflib"
)

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

type Change struct {
	Action             string
	Kind               string
	Name               string
	Patches            []*jsonPatch
	CurrentState       string
	DesiredState       string
	MaskedDesiredState string
}

type jsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (c *Change) ItemName() string {
	return kindToShortMapping[c.Kind] + "/" + c.Name
}

func (c *Change) JsonPatches(pretty bool) string {
	var b []byte
	if pretty {
		b, _ = json.MarshalIndent(c.Patches, "", "  ")
	} else {
		b, _ = json.Marshal(c.Patches)
	}
	return string(b)
}

func (c *Change) Diff(revealSecrets bool) string {
	var desired string
	if revealSecrets {
		desired = c.DesiredState
	} else {
		desired = c.MaskedDesiredState
	}
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(c.CurrentState),
		B:        difflib.SplitLines(desired),
		FromFile: "Current State (OpenShift cluster)",
		ToFile:   "Desired State (Processed template)",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

func (c *Change) addPatch(patch *jsonPatch) {
	c.Patches = append(c.Patches, patch)
	sort.Slice(c.Patches, func(i, j int) bool {
		return c.Patches[i].Path < c.Patches[j].Path
	})
}
