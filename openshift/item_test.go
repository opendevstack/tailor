package openshift

import (
	"bytes"
	"strings"
	"testing"
)

func TestImmutableFieldsEqual(t *testing.T) {
	remoteItem, err := getRoute([]byte("old.com"))
	if err != nil {
		t.Errorf("Did not get remote route.")
	}

	unchangedLocalItem, err := getRoute([]byte("old.com"))
	if err != nil {
		t.Errorf("Did not get local route.")
	}
	if unchangedLocalItem.YamlConfig() != remoteItem.YamlConfig() {
		t.Errorf("Local and remote route should be in sync.")
	}

	if !unchangedLocalItem.ImmutableFieldsEqual(remoteItem) {
		t.Errorf("Immutable field host should be the same.")
	}

	changedLocalItem, err := getRoute([]byte("new.com"))
	if err != nil {
		t.Errorf("Did not get local route.")
	}
	if changedLocalItem.YamlConfig() == remoteItem.YamlConfig() {
		t.Errorf("Local and remote route should have drift.")
	}

	if changedLocalItem.ImmutableFieldsEqual(remoteItem) {
		t.Errorf("Immutable field host should be different.")
	}
}

func TestDesiredConfig(t *testing.T) {
	localItem, err := getLocalDeploymentConfig()
	if err != nil {
		t.Errorf("Did not get local deployment config.")
	}
	remoteItem, err := getRemoteDeploymentConfig()
	if err != nil {
		t.Errorf("Did not get remote deployment config.")
	}

	if localItem.YamlConfig() != remoteItem.YamlConfig() {
		t.Errorf("Local and remote deployment config did not match.")
	}

	desiredConfig := localItem.DesiredConfig(remoteItem)

	if !strings.Contains(desiredConfig, "192.168.0.1:5000") {
		t.Errorf("Desired config did not contain image ref.")
	}
}

func getRoute(host []byte) (*ResourceItem, error) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: Route
  metadata:
    annotations: {}
    creationTimestamp: null
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
    wildcardPolicy: None
kind: List
metadata: {}
`)
	config := NewConfigFromList(bytes.Replace(byteList, []byte("HOST"), host, -1))
	filter := &ResourceFilter{
		Kinds: []string{"Route"},
		Name:  "",
		Label: "",
	}
	list := &ResourceList{Filter: filter}
	list.AppendItems(config)
	return list.GetItem("Route", "foo")
}

func getLocalDeploymentConfig() (*ResourceItem, error) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: DeploymentConfig
  metadata:
    creationTimestamp: null
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
        creationTimestamp: null
        labels:
          name: foo
      spec:
        containers:
        - image: bar/foo:latest
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
    triggers: []
kind: List
metadata: {}
`)
	config := NewConfigFromList(byteList)
	filter := &ResourceFilter{
		Kinds: []string{"DeploymentConfig"},
		Name:  "",
		Label: "",
	}
	list := &ResourceList{Filter: filter}
	list.AppendItems(config)
	return list.GetItem("DeploymentConfig", "foo")
}

func getRemoteDeploymentConfig() (*ResourceItem, error) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: DeploymentConfig
  metadata:
    creationTimestamp: null
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
        creationTimestamp: null
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
    triggers: []
kind: List
metadata: {}
`)
	config := NewConfigFromList(byteList)
	filter := &ResourceFilter{
		Kinds: []string{"DeploymentConfig"},
		Name:  "",
		Label: "",
	}
	list := &ResourceList{Filter: filter}
	list.AppendItems(config)
	return list.GetItem("DeploymentConfig", "foo")
}
