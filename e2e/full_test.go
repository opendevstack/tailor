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

	ocDiffBinary := getOcDiffBinary()

	export(t, ocDiffBinary)

	statusWithNoExpectedDrift(t, ocDiffBinary)

	// Create new resource
	fmt.Println("Create new template with one resource")
	cmBytes := []byte(
		`apiVersion: v1
kind: Template
metadata:
  creationTimestamp: null
  name: configmap
objects:
- apiVersion: v1
  data:
    bar: baz
  kind: ConfigMap
  metadata:
    creationTimestamp: null
    name: foo
`)
	ioutil.WriteFile("cm-template.yml", cmBytes, 0644)

	// Status -> expected to have one created resource
	cmd := exec.Command(ocDiffBinary, []string{"status"}...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	fmt.Println("Got status in test project (should show created resource)")
	if !strings.Contains(string(out), "to be created") {
		t.Fatalf("One resource should need to be created")
	}
	if strings.Contains(string(out), "to be updated") {
		t.Fatalf("No resources should need to be updated")
	}
	if strings.Contains(string(out), "to be deleted") {
		t.Fatalf("No resources should need to be deleted")
	}
	if !strings.Contains(string(out), "is in sync") {
		t.Fatalf("Some resources should be listed")
	}

	update(t, ocDiffBinary)
	statusWithNoExpectedDrift(t, ocDiffBinary)

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
	cmd = exec.Command(ocDiffBinary, []string{"status"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	fmt.Println("Got status in test project (should show updated resource)")
	if strings.Contains(string(out), "to be created") {
		t.Fatalf("No resources should need to be created")
	}
	if !strings.Contains(string(out), "to be updated") {
		t.Fatalf("One resource should need to be updated")
	}
	if strings.Contains(string(out), "to be deleted") {
		t.Fatalf("No resources should need to be deleted")
	}
	if !strings.Contains(string(out), "is in sync") {
		t.Fatalf("Some resources should be listed")
	}

	update(t, ocDiffBinary)
	statusWithNoExpectedDrift(t, ocDiffBinary)

	// Simulate manual change in cluster
	cmd = exec.Command("oc", []string{"patch", "cm/foo", "-p", "{\"data\": {\"bar\": \"baz\"}}"}...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not patch content of ConfigMap")
	}
	fmt.Println("Patched content of ConfigMap")

	// Status -> expected to have drift (updated resource)
	cmd = exec.Command(ocDiffBinary, []string{"status"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	fmt.Println("Got status in test project (should show updated resource)")
	if strings.Contains(string(out), "to be created") {
		t.Fatalf("No resources should need to be created")
	}
	if !strings.Contains(string(out), "to be updated") {
		t.Fatalf("One resource should need to be updated")
	}
	if strings.Contains(string(out), "to be deleted") {
		t.Fatalf("No resources should need to be deleted")
	}
	if !strings.Contains(string(out), "is in sync") {
		t.Fatalf("Some resources should be listed")
	}

	update(t, ocDiffBinary)
	statusWithNoExpectedDrift(t, ocDiffBinary)

	fmt.Println("Remove ConfigMap template")
	os.Remove("cm-template.yml")

	// Status -> expected to have drift (deleted resource)
	cmd = exec.Command(ocDiffBinary, []string{"status"}...)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	fmt.Println("Got status in test project (should show updated resource)")
	if strings.Contains(string(out), "to be created") {
		t.Fatalf("No resources should need to be created")
	}
	if strings.Contains(string(out), "to be updated") {
		t.Fatalf("No resources should need to be updated")
	}
	if !strings.Contains(string(out), "to be deleted") {
		t.Fatalf("One resource should need to be deleted")
	}
	if !strings.Contains(string(out), "is in sync") {
		t.Fatalf("Some resources should be listed")
	}

	update(t, ocDiffBinary)
	statusWithNoExpectedDrift(t, ocDiffBinary)
}

func update(t *testing.T, ocDiffBinary string) {
	cmd := exec.Command(ocDiffBinary, []string{"update", "--non-interactive"}...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not update test project")
	}
	fmt.Println("Updated test project")
}

func statusWithNoExpectedDrift(t *testing.T, ocDiffBinary string) {
	cmd := exec.Command(ocDiffBinary, []string{"status"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not get status in test project: %s", out)
	}
	fmt.Println("Got status in test project (should have no drift)")
	if strings.Contains(string(out), "to be created") {
		t.Fatalf("No resources should need to be created")
	}
	if strings.Contains(string(out), "to be updated") {
		t.Fatalf("No resources should need to be updated")
	}
	if strings.Contains(string(out), "to be deleted") {
		t.Fatalf("No resources should need to be deleted")
	}
	if !strings.Contains(string(out), "is in sync") {
		t.Fatalf("Some resources should be listed")
	}
}

func export(t *testing.T, ocDiffBinary string) {
	cmd := exec.Command(ocDiffBinary, []string{"export"}...)
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

func getOcDiffBinary() string {
	dir, _ := os.Getwd()
	return dir + "/../ocdiff-test"
}
