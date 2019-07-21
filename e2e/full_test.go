package e2e

import (
	"bytes"
	"fmt"
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
	fmt.Println("Create new template with one resource")
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
	ioutil.WriteFile("cm-template.yml", cmBytes, 0644)

	// Status -> expected to have one created resource
	cmd := exec.Command(tailorBinary, []string{"status", "--force"}...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	fmt.Println("Got status in test project (should show created resource)")

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
	fmt.Println("Got content of ConfigMap")
	if !strings.Contains(string(out), "bar: baz") {
		t.Fatalf("data should be 'bar: baz")
	}

	// Change content of local template
	fmt.Println("Change content of ConfigMap template")
	changedCmBytes := bytes.Replace(cmBytes, []byte("bar: baz"), []byte("bar: qux"), -1)
	ioutil.WriteFile("cm-template.yml", changedCmBytes, 0644)

	// Status -> expected to have drift (updated resource)
	cmd = exec.Command(tailorBinary, []string{"status", "--force"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	fmt.Println("Got status in test project (should show updated resource)")
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
	fmt.Println("Patched content of ConfigMap")

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

	fmt.Println("Remove ConfigMap template")
	os.Remove("cm-template.yml")

	// Status -> expected to have drift (deleted resource)
	cmd = exec.Command(tailorBinary, []string{"status", "--force"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	fmt.Println("Got status in test project (should show deleted resource)")
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
	fmt.Println("Updating test project")
	cmd := exec.Command(tailorBinary, []string{"update", "--non-interactive", "--force"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not update test project: %s", out)
	}
	fmt.Println("Updated test project")
}

func statusWithNoExpectedDrift(t *testing.T, tailorBinary string) {
	fmt.Println("Getting status with no expected drift")
	cmd := exec.Command(tailorBinary, []string{"status", "--force"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not get status in test project: %s", out)
	}
	fmt.Println("Got status in test project (should have no drift)")
	if !strings.Contains(string(out), "0 to create") {
		t.Fatalf("No resource should be to create")
	}
	if !strings.Contains(string(out), "0 to update") {
		t.Fatalf("No resource should be to update")
	}
	if !strings.Contains(string(out), "0 to delete") {
		t.Fatalf("No resource should be to delete")
	}
	if !strings.Contains(string(out), "in sync") {
		t.Fatalf("Some resources should be in sync")
	}
}

func export(t *testing.T, tailorBinary string) {
	cmd := exec.Command(tailorBinary, []string{"export", "--force"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not export resources in test project: %s", out)
	}
	ioutil.WriteFile("test-template.yml", out, 0644)
	fmt.Println("Resources in test project exported")
}

func setup(t *testing.T) {
	fmt.Println("Launching local cluster ...")
	cmd := exec.Command("oc", []string{"cluster", "up"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not launch local cluster: %s", out)
	}
	fmt.Println("Local cluster launched")

	cmd = exec.Command("oc", []string{"new-project", "test"}...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not create test project: %s", out)
	}
	fmt.Println("Test project created")

	os.MkdirAll("templates", os.ModePerm)
	os.Chdir("templates")
	fmt.Println("templates folder created")
}

func teardown(t *testing.T) {
	cmd := exec.Command("oc", []string{"delete", "project", "test"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not delete test project: %s", out)
	}
	fmt.Println("Test project deleted")

	cmd = exec.Command("oc", []string{"cluster", "down"}...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not shutdown local cluster: %s", out)
	}
	fmt.Println("Local cluster shut down")

	dir, _ := os.Getwd()
	os.Chdir(strings.TrimSuffix(dir, "/templates"))
	err = os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("Could not remove templates folder: %s", err)
	}
	fmt.Println("templates folder removed")
}

func getTailorBinary() string {
	dir, _ := os.Getwd()
	return dir + "/../tailor-test"
}
