package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"
)

func TestPartialScope(t *testing.T) {
	defer teardown(t)
	setup(t)

	ocDiffBinary := getOcDiffBinary()

	export(t, ocDiffBinary)

	statusWithNoExpectedDrift(t, ocDiffBinary)

	fmt.Println("Create new template with label app=foo")
	fooBytes := []byte(
		`apiVersion: v1
kind: Template
metadata:
  creationTimestamp: null
  name: configmap
objects:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    creationTimestamp: null
    name: foo
    labels:
      app: foo
  data:
    bar: baz
- apiVersion: v1
  kind: Service
  metadata:
    creationTimestamp: null
    labels:
      app: foo
    name: foo
  spec:
    ports:
    - name: web
      port: 80
      protocol: TCP
      targetPort: 8080
    selector:
      name: foo
    sessionAffinity: None
    type: ClusterIP
`)
	ioutil.WriteFile("foo-template.yml", fooBytes, 0644)

	fmt.Println("Create new template with label app=bar")
	barBytes := []byte(
		`apiVersion: v1
kind: Template
metadata:
  creationTimestamp: null
  name: configmap
objects:
- apiVersion: v1
  kind: ConfigMap
  metadata:
    creationTimestamp: null
    name: bar
    labels:
      app: bar
  data:
    bar: baz
- apiVersion: v1
  kind: Service
  metadata:
    creationTimestamp: null
    labels:
      app: bar
    name: bar
  spec:
    ports:
    - name: web
      port: 80
      protocol: TCP
      targetPort: 8080
    selector:
      name: bar
    sessionAffinity: None
    type: ClusterIP
`)
	ioutil.WriteFile("bar-template.yml", barBytes, 0644)

	update(t, ocDiffBinary)
	statusWithNoExpectedDrift(t, ocDiffBinary)

	partialStatusWithNoExpectedDrift(t, ocDiffBinary, "app=foo")
	partialStatusWithNoExpectedDrift(t, ocDiffBinary, "app=bar")

	// Change content of local template
	fmt.Println("Change content of ConfigMap template")
	changedFooBytes := bytes.Replace(fooBytes, []byte("bar: baz"), []byte("bar: qux"), -1)
	ioutil.WriteFile("foo-template.yml", changedFooBytes, 0644)

	// Status for app=foo -> expected to have drift (updated resource)
	cmd := exec.Command(ocDiffBinary, []string{"status", "-l", "app=foo"}...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Status command should have exited with 3")
	}
	fmt.Println("Got status for app=foo in test project (should show updated resource)")
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

	partialStatusWithNoExpectedDrift(t, ocDiffBinary, "app=bar")
}

func partialStatusWithNoExpectedDrift(t *testing.T, ocDiffBinary string, label string) {
	cmd := exec.Command(ocDiffBinary, []string{"status", "-l", label}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not get status for %s in test project", label)
	}
	fmt.Println("Got status for", label, "in test project (should have no drift)")
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
