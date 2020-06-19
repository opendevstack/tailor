package e2e

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestFullScope(t *testing.T) {
	testProjectName := setup(t)
	defer teardown(t, testProjectName)

	tailorBinary := getTailorBinary()

	runExport(t, tailorBinary, testProjectName)

	diffWithNoExpectedDrift(t, tailorBinary, testProjectName, []string{})

	// Create new resource
	t.Log("Create new template with one resource")
	cmBytes := []byte(
		`apiVersion: v1
kind: Template
metadata:
  name: configmap
objects:
- apiVersion: v1
  data:
    bar: baz
  kind: ConfigMap
  metadata:
    name: foo
`)
	err := ioutil.WriteFile("cm-template.yml", cmBytes, 0644)
	if err != nil {
		t.Fatalf("Fail to write file cm-template.yml: %s", err)
	}

	// Status -> expected to have one created resource
	cmd := exec.Command(tailorBinary, []string{"-n", testProjectName, "diff"}...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	t.Log("Got status in test project (should show created resource)")

	if !strings.Contains(string(out), "1 to create") {
		t.Fatalf("One resource should be to create")
	}
	if !strings.Contains(string(out), "0 to update") {
		t.Fatalf("No resource should be to update")
	}
	if !strings.Contains(string(out), "0 to delete") {
		t.Fatalf("No resource should be to delete")
	}
	if !strings.Contains(string(out), "in sync") {
		t.Fatalf("Some resources should be in synce")
	}

	runApply(t, tailorBinary, testProjectName, []string{})
	diffWithNoExpectedDrift(t, tailorBinary, testProjectName, []string{})

	// Check content of config map
	cmd = exec.Command("oc", []string{"-n", testProjectName, "get", "cm/foo", "-oyaml"}...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not get content of ConfigMap")
	}
	t.Log("Got content of ConfigMap")
	if !strings.Contains(string(out), "bar: baz") {
		t.Fatalf("data should be 'bar: baz")
	}

	// Change content of local template
	t.Log("Change content of ConfigMap template")
	changedCmBytes := bytes.Replace(cmBytes, []byte("bar: baz"), []byte("bar: qux"), -1)
	err = ioutil.WriteFile("cm-template.yml", changedCmBytes, 0644)
	if err != nil {
		t.Fatalf("Fail to write file cm-template.yml: %s", err)
	}

	// Status -> expected to have drift (updated resource)
	cmd = exec.Command(tailorBinary, []string{"-n", testProjectName, "diff"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	t.Log("Got status in", testProjectName, "project (should show updated resource)")
	if !strings.Contains(string(out), "0 to create") {
		t.Fatalf("No resource should be to create")
	}
	if !strings.Contains(string(out), "1 to update") {
		t.Fatalf("One resource should be to update")
	}
	if !strings.Contains(string(out), "0 to delete") {
		t.Fatalf("No resource should be to delete")
	}
	if !strings.Contains(string(out), "in sync") {
		t.Fatalf("Some resources should be in synce")
	}

	runApply(t, tailorBinary, testProjectName, []string{})
	diffWithNoExpectedDrift(t, tailorBinary, testProjectName, []string{})

	// Simulate manual change in cluster
	cmd = exec.Command("oc", []string{"-n", testProjectName, "patch", "cm/foo", "-p", "{\"data\": {\"bar\": \"baz\"}}"}...)
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not patch content of ConfigMap")
	}
	t.Log("Patched content of ConfigMap")

	// Status -> expected to have drift (updated resource)
	cmd = exec.Command(tailorBinary, []string{"-n", testProjectName, "diff"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	if !strings.Contains(string(out), "0 to create") {
		t.Fatalf("No resource should be to create")
	}
	if !strings.Contains(string(out), "1 to update") {
		t.Fatalf("One resource should be to update")
	}
	if !strings.Contains(string(out), "0 to delete") {
		t.Fatalf("No resource should be to delete")
	}
	if !strings.Contains(string(out), "in sync") {
		t.Fatalf("Some resources should be in synce")
	}

	runApply(t, tailorBinary, testProjectName, []string{})
	diffWithNoExpectedDrift(t, tailorBinary, testProjectName, []string{})

	t.Log("Remove ConfigMap template")
	os.Remove("cm-template.yml")

	// Status -> expected to have drift (deleted resource)
	cmd = exec.Command(tailorBinary, []string{"-n", testProjectName, "diff"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	t.Log("Got status in", testProjectName, "project (should show deleted resource)")
	if !strings.Contains(string(out), "0 to create") {
		t.Fatalf("No resource should be to create")
	}
	if !strings.Contains(string(out), "0 to update") {
		t.Fatalf("No resource should be to update")
	}
	if !strings.Contains(string(out), "1 to delete") {
		t.Fatalf("One resource should be to delete")
	}
	if !strings.Contains(string(out), "in sync") {
		t.Fatalf("Some resources should be in synce")
	}

	runApply(t, tailorBinary, testProjectName, []string{})
	diffWithNoExpectedDrift(t, tailorBinary, testProjectName, []string{})
}
