package openshift

import (
	"bytes"
	"testing"

	"github.com/ghodss/yaml"
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

func getConfigMap(annotations []byte) []byte {
	config := []byte(
		`apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app: bar
  annotations: ANNOTATIONS
  name: bar
data:
  bar: baz`)
	return bytes.Replace(config, []byte("ANNOTATIONS"), annotations, -1)
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

func getTemplateDeploymentConfig(tag []byte) []byte {
	config := []byte(
		`apiVersion: v1
kind: DeploymentConfig
metadata:
  name: foo
spec:
  replicas: 1
  selector:
    name: foo
  strategy:
    type: Recreate
  template:
    metadata:
      annotations: {}
      labels:
        name: foo
    spec:
      containers:
      - image: bar/foo:TAG
        imagePullPolicy: IfNotPresent
        name: foo
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: foo
      serviceAccountName: foo
      volumes: []
  test: false
  triggers:
  - type: ImageChange
    imageChangeParams: {}`)
	return bytes.Replace(config, []byte("TAG"), tag, -1)
}

func getPlatformDeploymentConfig() []byte {
	return []byte(
		`apiVersion: v1
kind: DeploymentConfig
metadata:
  name: foo
  annotations:
    original-values.tailor.io/spec.template.spec.containers.0.image: 'bar/foo:latest'
spec:
  replicas: 1
  selector:
    name: foo
  strategy:
    type: Recreate
  template:
    metadata:
      annotations: {}
      labels:
        name: foo
    spec:
      containers:
      - image: 192.168.0.1:5000/bar/foo@sha256:51ead8367892a487ca4a1ca7435fa418466901ca2842b777e15a12d0b470ab30
        imagePullPolicy: IfNotPresent
        name: foo
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: foo
      serviceAccountName: foo
      volumes: []
  test: false
  triggers:
  - type: ImageChange
    imageChangeParams:
      lastTriggeredImage: 127.0.0.1:5000/bar/foo@sha256:123`)
}
