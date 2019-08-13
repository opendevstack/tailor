package openshift

import (
	"reflect"
	"testing"
)

func TestAddCreateOrder(t *testing.T) {
	cs := &Changeset{}
	cDC := &Change{
		Action: "Create",
		Kind:   "DeploymentConfig",
	}
	cPVC := &Change{
		Action: "Create",
		Kind:   "PersistentVolumeClaim",
	}
	cs.Add(cPVC, cDC)
	if cs.Create[0].Kind != "PersistentVolumeClaim" {
		t.Errorf("PVC needs to be created before DC")
	}
}

func TestAddUpdateOrder(t *testing.T) {
	cs := &Changeset{}
	cDC := &Change{
		Action: "Update",
		Kind:   "DeploymentConfig",
	}
	cPVC := &Change{
		Action: "Update",
		Kind:   "PersistentVolumeClaim",
	}
	cs.Add(cPVC, cDC)
	if cs.Update[0].Kind != "PersistentVolumeClaim" {
		t.Errorf("PVC needs to be updated before DC")
	}
}

func TestAddDeleteOrder(t *testing.T) {
	cs := &Changeset{}
	cDC := &Change{
		Action: "Delete",
		Kind:   "DeploymentConfig",
	}
	cPVC := &Change{
		Action: "Delete",
		Kind:   "PersistentVolumeClaim",
	}
	cs.Add(cPVC, cDC)
	if cs.Delete[0].Kind != "DeploymentConfig" {
		t.Errorf("DC needs to be deleted before PVC")
	}
}

func TestConfigNoop(t *testing.T) {

	templateInput := []byte(
		`kind: List
metadata: {}
apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    labels:
      template: foo-template
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}`)

	platformInput := []byte(
		`kind: Template
metadata: {}
apiVersion: v1
objects:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    annotations:
      pv.kubernetes.io/bind-completed: "yes"
      pv.kubernetes.io/bound-by-controller: "yes"
      volume.beta.kubernetes.io/storage-provisioner: kubernetes.io/aws-ebs
    labels:
      template: foo-template
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
    volumeName: pvc-2150713e-3e20-11e8-aa60-0aad3152d0e6
  status: {}`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
	}
	changeset := getChangeset(t, filter, platformInput, templateInput, false, []string{})
	if !changeset.Blank() {
		//t.Errorf("Changeset is not blank, got %v", changeset.Update[0].JsonPatches(true))
		t.Fail()
	}
}

func TestConfigUpdate(t *testing.T) {

	templateInput := []byte(
		`kind: List
metadata: {}
apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: foo
    labels:
      app: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}`)

	platformInput := []byte(
		`kind: Template
metadata: {}
apiVersion: v1
objects:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: foo
    annotations:
      kubectl.kubernetes.io/last-applied-configuration: >
        {"apiVersion":"1"}
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
	}
	changeset := getChangeset(t, filter, platformInput, templateInput, false, []string{})
	if len(changeset.Update) != 1 {
		t.Errorf("Changeset.Update has %d items instead of 1", len(changeset.Update))
	}
}

func TestConfigIgnoredPaths(t *testing.T) {
	templateInput := []byte(
		`kind: List
apiVersion: v1
items:
- apiVersion: v1
  kind: BuildConfig
  metadata:
    name: foo
  spec:
    failedBuildsHistoryLimit: 5
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
      type: Generic`)

	platformInput := []byte(
		`kind: Template
apiVersion: v1
objects:
- apiVersion: v1
  kind: BuildConfig
  metadata:
    name: foo
  spec:
    failedBuildsHistoryLimit: 5
    output:
      to:
        kind: ImageStreamTag
        name: foo:abcdef
      imageLabels:
      - name: bar
        value: baz
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
      type: Generic`)

	filter := &ResourceFilter{
		Kinds: []string{"BuildConfig"},
	}
	changeset := getChangeset(t, filter, platformInput, templateInput, false, []string{"bc:/spec/output/to/name", "bc:/spec/output/imageLabels"})
	actualUpdates := len(changeset.Update)
	expectedUpdates := 0
	if actualUpdates != expectedUpdates {
		t.Errorf("Changeset.Update has %d items instead of %d", actualUpdates, expectedUpdates)
		for i, u := range changeset.Update {
			t.Errorf("Patchset Update#%d: %s", i, u.JsonPatches(true))
		}
	}
}

func TestConfigCreation(t *testing.T) {
	templateInput := []byte(
		`kind: List
metadata: {}
apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}`)

	platformInput := []byte(
		`kind: Template
metadata: {}
apiVersion: v1
objects:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: bar
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
	}
	changeset := getChangeset(t, filter, platformInput, templateInput, false, []string{})
	if len(changeset.Create) != 1 {
		t.Errorf("Changeset.Create is blank but should not be")
	}
}

func TestConfigDeletion(t *testing.T) {

	templateInput := []byte{}

	platformInput := []byte(
		`kind: Template
metadata: {}
apiVersion: v1
objects:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
	}
	changeset := getChangeset(t, filter, platformInput, templateInput, false, []string{})
	if len(changeset.Delete) != 1 {
		t.Errorf("Changeset.Delete is blank but should not be")
	}
}

func TestChangesFromEqual(t *testing.T) {
	currentItem := getItem(t, getBuildConfig(), "platform")
	desiredItem := getItem(t, getBuildConfig(), "template")
	_, err := calculateChanges(desiredItem, currentItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestChangesFromDifferent(t *testing.T) {
	currentItem := getItem(t, getBuildConfig(), "platform")
	desiredItem := getItem(t, getChangedBuildConfig(), "template")
	changes, err := calculateChanges(desiredItem, currentItem, []string{})
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
	changes, err := calculateChanges(unchangedTemplateItem, platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) > 1 || changes[0].Action != "Noop" {
		t.Errorf("Platform and template should be in sync, got %d change(s): %v", len(changes), changes[0])
	}

	changedTemplateItem := getItem(t, getRoute([]byte("new.com")), "template")
	changes, err = calculateChanges(changedTemplateItem, platformItem, []string{})
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
	changes, err := calculateChanges(templateItem, platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) > 1 || changes[0].Action != "Noop" {
		t.Errorf("Platform and template should be in sync, got %v", changes[0].JsonPatches(true))
	}

	changedTemplateItem := getItem(t, getTemplateDeploymentConfig([]byte("test")), "template")
	changes, err = calculateChanges(changedTemplateItem, platformItem, []string{})
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
	changes, err := calculateChanges(templateItem, platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) != 1 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatchOne := changes[0].Patches[0]
	actualPatchTwo := changes[0].Patches[1]
	expectedPatchOne := &jsonPatch{
		Op:    "add",
		Path:  "/metadata/annotations/foo",
		Value: "bar",
	}
	expectedPatchTwo := &jsonPatch{
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
	changes, err = calculateChanges(templateItem, platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	var actualPatch *jsonPatch
	var expectedPatch *jsonPatch
	if len(changes) > 1 || changes[0].Action != "Noop" {
		actualPatch = changes[0].Patches[0]
		t.Errorf("Platform and template should have no drift, got %v", actualPatch)
	}

	t.Log("> Adding a platform-managed annotation from the template")
	platformItem = getItem(t, getConfigMap([]byte("{foo: bar}")), "platform")
	templateItem = getItem(t, getConfigMap([]byte("{foo: bar}")), "template")
	changes, err = calculateChanges(templateItem, platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) != 1 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatch = changes[0].Patches[0]
	expectedPatch = &jsonPatch{
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
	changes, err = calculateChanges(templateItem, platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) == 0 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatch = changes[0].Patches[0]
	expectedPatch = &jsonPatch{
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
	changes, err = calculateChanges(templateItem, platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) == 0 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatch = changes[0].Patches[0]
	expectedPatch = &jsonPatch{
		Op:    "replace",
		Path:  "/metadata/annotations/foo",
		Value: "baz",
	}
	if !reflect.DeepEqual(actualPatch, expectedPatch) {
		t.Errorf("Got %v instead of %v", actualPatch, expectedPatch)
	}

	t.Log("> - Removing it")
	templateItem = getItem(t, getConfigMap([]byte("{}")), "template")
	changes, err = calculateChanges(templateItem, platformItem, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) != 1 {
		t.Errorf("Platform and template should have drift")
	}
	actualPatch = changes[0].Patches[0]
	expectedPatch = &jsonPatch{
		Op:   "remove",
		Path: "/metadata/annotations/foo",
	}
	if !reflect.DeepEqual(actualPatch, expectedPatch) {
		t.Errorf("Got %v instead of %v", actualPatch, expectedPatch)
	}
}

func getChangeset(t *testing.T, filter *ResourceFilter, platformInput, templateInput []byte, upsertOnly bool, ignoredPaths []string) *Changeset {
	platformBasedList, err := NewPlatformBasedResourceList(filter, platformInput)
	if err != nil {
		t.Error("Could not create platform based list:", err)
	}
	templateBasedList, err := NewTemplateBasedResourceList(filter, templateInput)
	if err != nil {
		t.Error("Could not create template based list:", err)
	}
	changeset, err := NewChangeset(platformBasedList, templateBasedList, upsertOnly, ignoredPaths)
	if err != nil {
		t.Error("Could not create changeset:", err)
	}
	return changeset
}
