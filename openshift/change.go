package openshift

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/opendevstack/tailor/cli"
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
	Action       string
	Kind         string
	Name         string
	Patches      []*JsonPatch
	CurrentState string
	DesiredState string
}

type JsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (c *Change) ItemName() string {
	return kindToShortMapping[c.Kind] + "/" + c.Name
}

func (c *Change) AddPatch(patch *JsonPatch) {
	c.Patches = append(c.Patches, patch)
	sort.Slice(c.Patches, func(i, j int) bool {
		return c.Patches[i].Path < c.Patches[j].Path
	})
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

func (c *Change) Diff() string {
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

func ocDelete(change *Change, compareOptions *cli.CompareOptions) error {
	kind := change.Kind
	name := change.Name
	fmt.Printf("Deleting %s/%s ... ", kind, name)
	args := []string{"delete", kind, name}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		"", // empty as name and selector is not allowed
	)
	_, errBytes, err := cli.RunCmd(cmd)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}
	return nil
}

func ocCreate(change *Change, compareOptions *cli.CompareOptions) error {
	kind := change.Kind
	name := change.Name
	config := change.DesiredState
	fmt.Printf("Creating %s/%s ... ", kind, name)
	args := []string{"create", "-f", "-"}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		compareOptions.Selector,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, config)
	}()
	_, errBytes, err := cli.RunCmd(cmd)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}

	return nil
}

func ocPatch(change *Change, compareOptions *cli.CompareOptions) error {
	kind := change.Kind
	name := change.Name

	j := change.JsonPatches(false)

	fmt.Printf("Patching %s/%s ... ", kind, name)

	args := []string{"patch", kind + "/" + name, "--type=json", "--patch", j}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		"", // empty as name and selector is not allowed
	)
	_, errBytes, err := cli.RunCmd(cmd)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}
	return nil
}
