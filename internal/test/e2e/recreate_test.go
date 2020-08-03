package e2e

import (
	"io/ioutil"
	"testing"
)

func TestRecreate(t *testing.T) {
	testProjectName := setup(t, true)
	defer teardown(t, testProjectName, true)

	tailorBinary := getTailorBinary()

	// Create new resource
	t.Log("Create new template with one resource")
	templateBytes := []byte(
		`apiVersion: v1
kind: Template
objects:
- apiVersion: v1
  kind: Route
  metadata:
    labels:
      app: foo-route
    name: foo
  spec:
    host: foo.example.com
    tls:
      insecureEdgeTerminationPolicy: Redirect
      termination: edge
    to:
      kind: Service
      name: foo
      weight: 100
    wildcardPolicy: None
`)
	err := ioutil.WriteFile("route-template.yml", templateBytes, 0644)
	if err != nil {
		t.Fatalf("Fail to write file route-template.yml: %s", err)
	}

	runApply(t, tailorBinary, testProjectName, []string{"--selector", "app=foo-route"})

	templateBytes = []byte(
		`apiVersion: v1
kind: Template
objects:
- apiVersion: v1
  kind: Route
  metadata:
    labels:
      app: foo-route
    name: foo
  spec:
    host: foobar.example.com
    tls:
      insecureEdgeTerminationPolicy: Redirect
      termination: edge
    to:
      kind: Service
      name: foo
      weight: 100
    wildcardPolicy: None
`)
	err = ioutil.WriteFile("route-template.yml", templateBytes, 0644)
	if err != nil {
		t.Fatalf("Fail to write file route-template.yml: %s", err)
	}

	runApply(t, tailorBinary, testProjectName, []string{"--selector", "app=foo-route", "--allow-recreate"})
}
