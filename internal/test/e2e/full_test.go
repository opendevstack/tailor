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
	defer teardown(t)
	setup(t)

	tailorBinary := getTailorBinary()

	export(t, tailorBinary)

	statusWithNoExpectedDrift(t, tailorBinary)

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
	cmd := exec.Command(tailorBinary, []string{"status", "--force"}...)
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

	update(t, tailorBinary)
	statusWithNoExpectedDrift(t, tailorBinary)

	// Check content of config map
	cmd = exec.Command("oc", []string{"get", "cm/foo", "-oyaml"}...)
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
	cmd = exec.Command(tailorBinary, []string{"status", "--force"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	t.Log("Got status in test project (should show updated resource)")
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

	update(t, tailorBinary)
	statusWithNoExpectedDrift(t, tailorBinary)

	// Simulate manual change in cluster
	cmd = exec.Command("oc", []string{"patch", "cm/foo", "-p", "{\"data\": {\"bar\": \"baz\"}}"}...)
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not patch content of ConfigMap")
	}
	t.Log("Patched content of ConfigMap")

	// Status -> expected to have drift (updated resource)
	cmd = exec.Command(tailorBinary, []string{"status", "--force"}...)
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

	update(t, tailorBinary)
	statusWithNoExpectedDrift(t, tailorBinary)

	t.Log("Remove ConfigMap template")
	os.Remove("cm-template.yml")

	// Status -> expected to have drift (deleted resource)
	cmd = exec.Command(tailorBinary, []string{"status", "--force"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	t.Log("Got status in test project (should show deleted resource)")
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

	update(t, tailorBinary)
	statusWithNoExpectedDrift(t, tailorBinary)
}

func update(t *testing.T, tailorBinary string) {
	t.Log("Updating test project")
	cmd := exec.Command(tailorBinary, []string{"update", "--non-interactive", "--force"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not update test project: %s", out)
	}
	t.Log("Updated test project")
}

func statusWithNoExpectedDrift(t *testing.T, tailorBinary string) {
	t.Log("Getting status ...")
	cmd := exec.Command(tailorBinary, []string{"status", "--force"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not get status in test project: %s", out)
	}
	t.Log("Got status in test project (should have no drift)")
	if !strings.Contains(string(out), "0 to create") {
		t.Fatalf("No resource should be to create. Got: %s", out)
	}
	if !strings.Contains(string(out), "0 to update") {
		t.Fatalf("No resource should be to update. Got: %s", out)
	}
	if !strings.Contains(string(out), "0 to delete") {
		t.Fatalf("No resource should be to delete. Got: %s", out)
	}
	if !strings.Contains(string(out), "in sync") {
		t.Fatalf("Some resources should be in sync. Got: %s", out)
	}
}

func export(t *testing.T, tailorBinary string) {
	cmd := exec.Command(tailorBinary, []string{"export", "--force"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not export resources in test project: %s", out)
	}
	err = ioutil.WriteFile("test-template.yml", out, 0644)
	if err != nil {
		t.Fatalf("Fail to write file cm-template.yml: %s", err)
	}
	t.Log("Resources in test project exported")
}

var shutdownLater bool

func setup(t *testing.T) {
	t.Log("SETUP: Checking for local cluster ...")
	cmd := exec.Command("oc", []string{"whoami"}...)
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Log("SETUP: Local cluster running ...")
		shutdownLater = false
	} else {
		shutdownLater = true
		launchLocalCluster(t)
	}
	makeTestProject(t)
	makeTemplateFolder(t)
}

func teardown(t *testing.T) {
	deleteTestProject(t)
	shutdownLocalCluster(t)
	cleanupTemplateFolder(t)
}

func getTailorBinary() string {
	dir, _ := os.Getwd()
	return dir + "/../tailor-test"
}

func launchLocalCluster(t *testing.T) {
	t.Log("SETUP: Launching local cluster ...")
	cmd := exec.Command("oc", []string{"cluster", "up"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SETUP: Could not launch local cluster: %s", out)
	}
	t.Log("SETUP: Local cluster launched")
}

func shutdownLocalCluster(t *testing.T) {
	if shutdownLater {
		cmd := exec.Command("oc", []string{"cluster", "down"}...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("TEARDOWN: Could not shutdown local cluster: %s", out)
		}
		t.Log("TEARDOWN: Local cluster shut down")
	}
}

func deleteTestProject(t *testing.T) {
	cmd := exec.Command("oc", []string{"delete", "project", "test"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("TEARDOWN: Could not delete test project: %s", out)
	}
	t.Log("TEARDOWN: Test project deleted")
}

func makeTestProject(t *testing.T) {
	cmd := exec.Command("oc", []string{"new-project", "test"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SETUP: Could not create test project: %s", out)
	}
	t.Log("SETUP: Test project created")
}

func makeTemplateFolder(t *testing.T) {
	err := os.MkdirAll("templates", os.ModePerm)
	if err != nil {
		t.Fatalf("SETUP: Fail to mkdir templates: %s", err)
	}
	err = os.Chdir("templates")
	if err != nil {
		t.Fatalf("SETUP: Fail to chdir templates: %s", err)
	}
	t.Log("SETUP: templates folder created")
}

func cleanupTemplateFolder(t *testing.T) {
	dir, _ := os.Getwd()
	err := os.Chdir(strings.TrimSuffix(dir, "/templates"))
	if err != nil {
		t.Fatalf("TEARDOWN: Fail to chdir templates: %s", err)
	}
	err = os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("TEARDOWN: Could not remove templates folder: %s", err)
	}
	t.Log("TEARDOWN: templates folder removed")
}
