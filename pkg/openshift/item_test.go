package openshift

import (
	"bytes"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/internal/test/helper"
)

func TestNewResourceItem(t *testing.T) {
	item := getItem(t, getBuildConfig(), "template")
	if item.Kind != "BuildConfig" {
		t.Errorf("Kind is %s but should be BuildConfig", item.Kind)
	}
	if item.Name != "foo" {
		t.Errorf("Name is %s but should be foo", item.Name)
	}
	if item.Labels["app"] != "foo" {
		t.Errorf("Label app is %s but should be foo", item.Labels["app"])
	}
}

func getPlatformItem(t *testing.T, filename string) *ResourceItem {
	return getItem(t, helper.ReadFixtureFile(t, filename), "platform")
}

func getTemplateItem(t *testing.T, filename string) *ResourceItem {
	return getItem(t, helper.ReadFixtureFile(t, filename), "template")
}

func getItem(t *testing.T, input []byte, source string) *ResourceItem {
	var f interface{}
	err := yaml.Unmarshal(input, &f)
	if err != nil {
		t.Fatalf("Could not umarshal yaml: %v", err)
	}
	m := f.(map[string]interface{})
	item, err := NewResourceItem(m, source)
	if err != nil {
		t.Errorf("Could not create item: %v", err)
	}
	return item
}

func getBuildConfig() []byte {
	return []byte(
		`apiVersion: v1
kind: BuildConfig
metadata:
  annotations: {}
  labels:
    app: foo
  name: foo
spec:
  failedBuildsHistoryLimit: 5
  nodeSelector: null
  output:
    to:
      kind: ImageStreamTag
      name: foo:latest
  postCommit: {}
  resources: {}
  runPolicy: Serial
  source:
    binary: {}
    type: Binary
  strategy:
    dockerStrategy: {}
    type: Docker
  successfulBuildsHistoryLimit: 5
  triggers:
  - generic:
      secret: password
    type: Generic
  - imageChange: {}
    type: ImageChange
  - type: ConfigChange`)
}

func getChangedBuildConfig() []byte {
	return []byte(
		`apiVersion: v1
kind: BuildConfig
metadata:
  annotations:
    foo: bar
  name: foo
spec:
  failedBuildsHistoryLimit: 8
  nodeSelector: null
  output:
    to:
      kind: ImageStreamTag
      name: foo:experiment
  postCommit: {}
  resources: {}
  runPolicy: Serial
  source:
    binary: {}
    type: Binary
  strategy:
    dockerStrategy: {}
    type: Docker
  successfulBuildsHistoryLimit: 5
  triggers:
  - imageChange: {}
    type: ImageChange
  - type: ConfigChange`)
}

func getRoute(host []byte) []byte {
	config := []byte(
		`apiVersion: v1
kind: Route
metadata:
  annotations: {}
  name: foo
spec:
  host: HOST
  tls:
    insecureEdgeTerminationPolicy: Redirect
    termination: edge
  to:
    kind: Service
    name: foo
    weight: 100
  wildcardPolicy: None`)

	return bytes.Replace(config, []byte("HOST"), host, -1)
}
