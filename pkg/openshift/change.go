package openshift

import (
	"encoding/json"
	"sort"

	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/utils"
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

// Change is a description of a drift between current and desired state, and
// the required patches to bring them back in sync.
type Change struct {
	Action       string
	Kind         string
	Name         string
	Patches      []*jsonPatch
	CurrentState string
	DesiredState string
}

type jsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type jsonPatches []*jsonPatch

func (jp jsonPatch) Pretty() string {
	var b []byte
	b, _ = json.MarshalIndent(jp, "", "  ")
	return string(b)
}

// NewChange creates a new change for given template/platform item.
func NewChange(templateItem *ResourceItem, platformItem *ResourceItem, comparison map[string]*jsonPatch) *Change {
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

	return c
}

// ItemName returns the kind/name of the resource the change relates to.
func (c *Change) ItemName() string {
	return kindToShortMapping[c.Kind] + "/" + c.Name
}

// PrettyJSONPatches prints the JSON patches in pretty form (with indentation).
func (c *Change) PrettyJSONPatches() string {
	var b []byte
	b, _ = json.MarshalIndent(c.Patches, "", "  ")
	return string(b)
}

// JSONPatches prints the JSON patches in plain form.
func (c *Change) JSONPatches() string {
	var b []byte
	b, _ = json.Marshal(c.Patches)
	return string(b)
}

// Diff returns a unified diff text for the change.
func (c *Change) Diff(revealSecrets bool) string {
	if len(c.Patches) > 0 && c.patchesAreTailorInternalOnly() {
		return "Only annotations used by Tailor internally differ. Updating the resource is recommended, but not required. Use --diff=json to see the exact changes.\n"
	} else if c.isSecret() && !revealSecrets {
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

func (c *Change) addPatch(patch *jsonPatch) {
	c.Patches = append(c.Patches, patch)
	sort.Slice(c.Patches, func(i, j int) bool {
		return c.Patches[i].Path < c.Patches[j].Path
	})
}

func (c *Change) patchesAreTailorInternalOnly() bool {
	internalPaths := []string{tailorAppliedConfigAnnotationPath, tailorManagedAnnotationPath}
	for _, p := range c.Patches {
		if !utils.Includes(internalPaths, p.Path) {
			return false
		}
	}
	return true
}

func (c *Change) isSecret() bool {
	return kindToShortMapping[c.Kind] == "secret"
}
