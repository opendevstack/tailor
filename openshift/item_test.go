package openshift

import (
	"bytes"
	"reflect"
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

func TestChangesFromEqual(t *testing.T) {
	currentItem := getItem(t, getBuildConfig(), "platform")
	desiredItem := getItem(t, getBuildConfig(), "template")
	desiredItem.ChangesFrom(currentItem, []string{})
}

func TestChangesFromDifferent(t *testing.T) {
	currentItem := getItem(t, getBuildConfig(), "platform")
	desiredItem := getItem(t, getChangedBuildConfig(), "template")
	changes, err := desiredItem.ChangesFrom(currentItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	change := changes[0]
	if len(change.Patches) != 11 {
		t.Errorf("Got %d instead of %d changes: %s", len(change.Patches), 11, change.JsonPatches(true))
	}
}

func TestChangesFromImmutableFields(t *testing.T) {
	platformItem := getItem(t, getRoute([]byte("old.com")), "platform")

	unchangedTemplateItem := getItem(t, getRoute([]byte("old.com")), "template")
	changes, err := unchangedTemplateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) > 1 || changes[0].Action != "Noop" {
		t.Errorf("Platform and template should be in sync, got %d change(s): %v", len(changes), changes[0])
	}

	changedTemplateItem := getItem(t, getRoute([]byte("new.com")), "template")
	changes, err = changedTemplateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) == 0 {
		t.Errorf("Platform and template should have drift.")
	}
}

func TestChangesFromPlatformModifiedFields(t *testing.T) {
	platformItem := getItem(t, getPlatformDeploymentConfig(), "platform")
	templateItem := getItem(t, getTemplateDeploymentConfig([]byte("latest")), "template")
	changes, err := templateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) > 1 || changes[0].Action != "Noop" {
		t.Errorf("Platform and template should be in sync, got %v", changes[0].JsonPatches(true))
	}

	changedTemplateItem := getItem(t, getTemplateDeploymentConfig([]byte("test")), "template")
	changes, err = changedTemplateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) != 1 {
		t.Errorf("Platform and template should have drift for image field")
	}

	patchAnnotation := changes[0].Patches[0]
	if patchAnnotation.Op != "replace" {
		t.Errorf("Got op %s instead of replace", patchAnnotation.Op)
	}
	if patchAnnotation.Path != "/metadata/annotations/original-values.tailor.io~1spec.template.spec.containers.0.image" {
		t.Errorf("Got path %s instead of /metadata/annotations/original-values.tailor.io~1spec.template.spec.containers.0.image", patchAnnotation.Path)
	}
	if patchAnnotation.Value != "bar/foo:test" {
		t.Errorf("Got op %s instead of bar/foo:test", patchAnnotation.Value)
	}
	patchImage := changes[0].Patches[1]
	if patchImage.Op != "replace" {
		t.Errorf("Got op %s instead of replace", patchImage.Op)
	}
	if patchImage.Path != "/spec/template/spec/containers/0/image" {
		t.Errorf("Got path %s instead of /spec/template/spec/containers/0/image", patchImage.Path)
	}
	if patchImage.Value != "bar/foo:test" {
		t.Errorf("Got op %s instead of bar/foo:test", patchImage.Value)
	}
}

func TestChangesFromAnnotationFields(t *testing.T) {
	t.Log("> Adding an annotation in the template")
	platformItem := getItem(t, getConfigMap([]byte("{}")), "platform")
	templateItem := getItem(t, getConfigMap([]byte("{foo: bar}")), "template")
	changes, err := templateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) != 1 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatchOne := changes[0].Patches[0]
	actualPatchTwo := changes[0].Patches[1]
	expectedPatchOne := &JsonPatch{
		Op:    "add",
		Path:  "/metadata/annotations/foo",
		Value: "bar",
	}
	expectedPatchTwo := &JsonPatch{
		Op:    "add",
		Path:  "/metadata/annotations/managed-annotations.tailor.opendevstack.org",
		Value: "foo",
	}
	if !reflect.DeepEqual(actualPatchOne, expectedPatchOne) {
		t.Errorf("Got %v instead of %v", actualPatchOne, expectedPatchOne)
	}
	if !reflect.DeepEqual(actualPatchTwo, expectedPatchTwo) {
		t.Errorf("Got %v instead of %v", actualPatchTwo, expectedPatchTwo)
	}

	t.Log("> Having a platform-managed annotation")
	platformItem = getItem(t, getConfigMap([]byte("{foo: bar}")), "platform")
	templateItem = getItem(t, getConfigMap([]byte("{}")), "template")
	changes, err = templateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	var actualPatch *JsonPatch
	var expectedPatch *JsonPatch
	if len(changes) > 1 || changes[0].Action != "Noop" {
		actualPatch = changes[0].Patches[0]
		t.Errorf("Platform and template should have no drift, got %v", actualPatch)
	}

	t.Log("> Adding a platform-managed annotation from the template")
	platformItem = getItem(t, getConfigMap([]byte("{foo: bar}")), "platform")
	templateItem = getItem(t, getConfigMap([]byte("{foo: bar}")), "template")
	changes, err = templateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) != 1 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatch = changes[0].Patches[0]
	expectedPatch = &JsonPatch{
		Op:    "add",
		Path:  "/metadata/annotations/managed-annotations.tailor.opendevstack.org",
		Value: "foo",
	}
	if !reflect.DeepEqual(actualPatch, expectedPatch) {
		t.Errorf("Got %v instead of %v", actualPatch, expectedPatch)
	}

	t.Log("> Changing a platform-managed annotation from the template")
	platformItem = getItem(t, getConfigMap([]byte("{foo: bar}")), "platform")
	templateItem = getItem(t, getConfigMap([]byte("{foo: baz}")), "template")
	changes, err = templateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) == 0 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatch = changes[0].Patches[0]
	expectedPatch = &JsonPatch{
		Op:    "replace",
		Path:  "/metadata/annotations/foo",
		Value: "baz",
	}
	if !reflect.DeepEqual(actualPatch, expectedPatch) {
		t.Errorf("Got %v instead of %v", actualPatch, expectedPatch)
	}

	t.Log("> Managed annotation")
	c := getConfigMap([]byte(`
    managed-annotations.tailor.opendevstack.org: foo
    foo: bar`))
	platformItem = getItem(t, c, "platform")

	t.Log("> - Modifying it")
	templateItem = getItem(t, getConfigMap([]byte("{foo: baz}")), "template")
	changes, err = templateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) == 0 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatch = changes[0].Patches[0]
	expectedPatch = &JsonPatch{
		Op:    "replace",
		Path:  "/metadata/annotations/foo",
		Value: "baz",
	}
	if !reflect.DeepEqual(actualPatch, expectedPatch) {
		t.Errorf("Got %v instead of %v", actualPatch, expectedPatch)
	}

	t.Log("> - Removing it")
	templateItem = getItem(t, getConfigMap([]byte("{}")), "template")
	changes, err = templateItem.ChangesFrom(platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) != 1 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatch = changes[0].Patches[0]
	expectedPatch = &JsonPatch{
		Op:   "remove",
		Path: "/metadata/annotations/foo",
	}
	if !reflect.DeepEqual(actualPatch, expectedPatch) {
		t.Errorf("Got %v instead of %v", actualPatch, expectedPatch)
	}
}

func getItem(t *testing.T, input []byte, source string) *ResourceItem {
	var f interface{}
	yaml.Unmarshal(input, &f)
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
