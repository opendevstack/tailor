package openshift

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opendevstack/tailor/internal/test/helper"
)

func TestNewChangesetCreationOfResources(t *testing.T) {
	tests := map[string]struct {
		templateFixture string
		expectedGolden  string
	}{
		"Without annotations": {
			templateFixture: "is.yml",
			expectedGolden:  "is.yml",
		},
		"With annotations": {
			templateFixture: "is-annotation.yml",
			expectedGolden:  "is-annotation.yml",
		},
		"With image reference": {
			templateFixture: "dc.yml",
			expectedGolden:  "dc.yml",
		},
		"With image reference and annotation": {
			templateFixture: "dc-annotation.yml",
			expectedGolden:  "dc-annotation.yml",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filter, err := NewResourceFilter("", "", "")
			if err != nil {
				t.Fatal(err)
			}
			platformBasedList, err := NewPlatformBasedResourceList(
				filter,
				[]byte(""), // empty to ensure creation of resource
			)
			if err != nil {
				t.Fatal(err)
			}
			templateBasedList, err := NewTemplateBasedResourceList(
				filter,
				helper.ReadFixtureFile(t, "templates/"+tc.templateFixture),
			)
			if err != nil {
				t.Fatal(err)
			}
			upsertOnly := false
			allowRecreate := false
			preservePaths := []string{}
			cs, err := NewChangeset(
				platformBasedList,
				templateBasedList,
				upsertOnly,
				allowRecreate,
				preservePaths,
			)
			if err != nil {
				t.Fatal(err)
			}
			createChanges := cs.Create
			numberOfCreateChanges := len(createChanges)
			if numberOfCreateChanges != 1 {
				t.Fatalf("Expected one creation change, got: %d", numberOfCreateChanges)
			}
			createChange := createChanges[0]
			want := string(helper.ReadGoldenFile(t, "desired-state/"+tc.expectedGolden))
			got := createChange.DesiredState
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Desired state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCalculateChangesManagedAnnotations(t *testing.T) {

	tests := map[string]struct {
		platformFixture        string
		templateFixture        string
		expectedAction         string
		expectedDiffGoldenFile string
	}{
		"Without annotations": {
			platformFixture: "is-platform",
			templateFixture: "is-template",
			expectedAction:  "Noop",
		},
		"Present in template, not in platform": {
			platformFixture:        "is-platform",
			templateFixture:        "is-template-annotation",
			expectedAction:         "Update",
			expectedDiffGoldenFile: "present-in-template-not-in-platform",
		},
		"Present in platform, not in template": {
			platformFixture:        "is-platform-annotation",
			templateFixture:        "is-template",
			expectedAction:         "Update",
			expectedDiffGoldenFile: "present-in-platform-not-in-template",
		},
		"Present in both": {
			platformFixture: "is-platform-annotation",
			templateFixture: "is-template-annotation",
			expectedAction:  "Noop",
		},
		"Present in platform, changed in template": {
			platformFixture:        "is-platform-annotation",
			templateFixture:        "is-template-annotation-changed",
			expectedAction:         "Update",
			expectedDiffGoldenFile: "present-in-platform-changed-in-template",
		},
		"Present in platform, different key in template": {
			platformFixture:        "is-platform-annotation",
			templateFixture:        "is-template-different-annotation",
			expectedAction:         "Update",
			expectedDiffGoldenFile: "present-in-platform-different-key-in-template",
		},
		"Unmanaged in platform added to template": {
			platformFixture: "is-platform-unmanaged",
			templateFixture: "is-template-annotation",
			expectedAction:  "Noop",
		},
		"Unmanaged in platform, none in template": {
			platformFixture: "is-platform-unmanaged",
			templateFixture: "is-template",
			expectedAction:  "Noop",
		},
		"Unmanaged in platform, none in template, and other change in template": {
			platformFixture:        "is-platform-unmanaged",
			templateFixture:        "is-template-other-change",
			expectedAction:         "Update",
			expectedDiffGoldenFile: "unmanaged-in-platform-none-in-template-other-change-in-template",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			platformItem := getPlatformItem(t, "item-managed-annotations/"+tc.platformFixture+".yml")
			templateItem := getTemplateItem(t, "item-managed-annotations/"+tc.templateFixture+".yml")
			changes, err := calculateChanges(templateItem, platformItem, []string{}, true)
			if err != nil {
				t.Fatal(err)
			}
			if len(changes) != 1 {
				t.Fatalf("Expected 1 change, got: %d", len(changes))
			}
			actualChange := changes[0]
			if actualChange.Action != tc.expectedAction {
				t.Fatalf("Expected change action to be: %s, got: %s", tc.expectedAction, actualChange.Action)
			}
			if len(tc.expectedDiffGoldenFile) > 0 {
				want := strings.TrimSpace(getGoldenDiff(t, "item-managed-annotations", tc.expectedDiffGoldenFile+".txt"))
				got := strings.TrimSpace(actualChange.Diff(true))
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("Change diff mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCalculateChangesAppliedConfiguration(t *testing.T) {

	tests := map[string]struct {
		platformFixture string
		templateFixture string
		expectedAction  string
	}{
		"Without annotation in platform": {
			platformFixture: "dc-platform",
			templateFixture: "dc-template",
			expectedAction:  "Update",
		},
		"With annotation in platform": {
			platformFixture: "dc-platform-annotation-other",
			templateFixture: "dc-template",
			expectedAction:  "Update",
		},
		"Present in platform": {
			platformFixture: "dc-platform-annotation-applied",
			templateFixture: "dc-template",
			expectedAction:  "Noop",
		},
		"Present in platform, changed in template": {
			platformFixture: "dc-platform-annotation-applied",
			templateFixture: "dc-template-changed",
			expectedAction:  "Update",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			platformItem := getPlatformItem(t, "item-applied-config/"+tc.platformFixture+".yml")
			templateItem := getTemplateItem(t, "item-applied-config/"+tc.templateFixture+".yml")
			changes, err := calculateChanges(templateItem, platformItem, []string{}, true)
			if err != nil {
				t.Fatal(err)
			}
			if len(changes) != 1 {
				t.Fatalf("Expected 1 change, got: %d", len(changes))
			}
			actualChange := changes[0]
			if actualChange.Action != tc.expectedAction {
				t.Fatalf("Expected change action to be: %s, got: %s", tc.expectedAction, actualChange.Action)
			}
		})
	}
}

func TestCalculateChangesOmittedFields(t *testing.T) {

	tests := map[string]struct {
		platformFixture        string
		templateFixture        string
		expectedAction         string
		expectedDiffGoldenFile string
	}{
		"Rolebinding with legacy fields": {
			platformFixture:        "rolebinding-platform",
			templateFixture:        "rolebinding-template",
			expectedAction:         "Update",
			expectedDiffGoldenFile: "rolebinding-changed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			platformItem := getPlatformItem(t, "item-omitted-fields/"+tc.platformFixture+".yml")
			templateItem := getTemplateItem(t, "item-omitted-fields/"+tc.templateFixture+".yml")
			changes, err := calculateChanges(templateItem, platformItem, []string{}, true)
			if err != nil {
				t.Fatal(err)
			}
			if len(changes) != 1 {
				t.Fatalf("Expected 1 change, got: %d", len(changes))
			}
			actualChange := changes[0]
			if actualChange.Action != tc.expectedAction {
				t.Fatalf("Expected change action to be: %s, got: %s", tc.expectedAction, actualChange.Action)
			}
			if len(tc.expectedDiffGoldenFile) > 0 {
				want := strings.TrimSpace(getGoldenDiff(t, "item-omitted-fields", tc.expectedDiffGoldenFile+".txt"))
				got := strings.TrimSpace(actualChange.Diff(true))
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("Change diff mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestEmptyValuesDoNotCauseDrift(t *testing.T) {
	platformItem := getPlatformItem(t, "empty-values/bc-platform.yml")
	templateItem := getTemplateItem(t, "empty-values/bc-template.yml")
	changes, err := calculateChanges(templateItem, platformItem, []string{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("Expected 1 change, got: %d", len(changes))
	}
	actualChange := changes[0]
	expectedAction := "Noop"
	if actualChange.Action != expectedAction {
		t.Fatalf("Expected change action to be: %s, got: %s. Diff was: %s", expectedAction, actualChange.Action, actualChange.Diff(false))
	}
}

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
	changeset := getChangeset(t, filter, platformInput, templateInput, false, true, []string{})
	if !changeset.Blank() {
		t.Fatalf("Changeset is not blank!")
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
	changeset := getChangeset(t, filter, platformInput, templateInput, false, true, []string{})
	if len(changeset.Update) != 1 {
		t.Errorf("Changeset.Update has %d items instead of 1", len(changeset.Update))
	}
}

func TestConfigPreservePaths(t *testing.T) {
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
	changeset := getChangeset(t, filter, platformInput, templateInput, false, true, []string{"bc:/spec/output/to/name", "bc:/spec/output/imageLabels"})
	actualUpdates := len(changeset.Update)
	expectedUpdates := 0
	if actualUpdates != expectedUpdates {
		t.Errorf("Changeset.Update has %d items instead of %d", actualUpdates, expectedUpdates)
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
	changeset := getChangeset(t, filter, platformInput, templateInput, false, true, []string{})
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
	changeset := getChangeset(t, filter, platformInput, templateInput, false, true, []string{})
	if len(changeset.Delete) != 1 {
		t.Errorf("Changeset.Delete is blank but should not be")
	}
}

func TestCalculateChangesEqual(t *testing.T) {
	currentItem := getItem(t, getBuildConfig(), "platform")
	desiredItem := getItem(t, getBuildConfig(), "template")
	_, err := calculateChanges(desiredItem, currentItem, []string{}, true)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestCalculateChangesImmutableFields(t *testing.T) {
	platformItem := getItem(t, getRoute([]byte("old.com")), "platform")

	unchangedTemplateItem := getItem(t, getRoute([]byte("old.com")), "template")
	changes, err := calculateChanges(unchangedTemplateItem, platformItem, []string{}, true)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) > 1 || changes[0].Action != "Noop" {
		t.Errorf("Platform and template should be in sync, got %d change(s): %v", len(changes), changes[0])
	}

	changedTemplateItem := getItem(t, getRoute([]byte("new.com")), "template")
	changes, err = calculateChanges(changedTemplateItem, platformItem, []string{}, true)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changes) == 0 {
		t.Errorf("Platform and template should have drift.")
	}
}

func getChangeset(t *testing.T, filter *ResourceFilter, platformInput, templateInput []byte, upsertOnly bool, allowRecreate bool, preservePaths []string) *Changeset {
	platformBasedList, err := NewPlatformBasedResourceList(filter, platformInput)
	if err != nil {
		t.Error("Could not create platform based list:", err)
	}
	templateBasedList, err := NewTemplateBasedResourceList(filter, templateInput)
	if err != nil {
		t.Error("Could not create template based list:", err)
	}
	changeset, err := NewChangeset(platformBasedList, templateBasedList, upsertOnly, allowRecreate, preservePaths)
	if err != nil {
		t.Error("Could not create changeset:", err)
	}
	return changeset
}

func getGoldenDiff(t *testing.T, folder string, filename string) string {
	b := helper.ReadGoldenFile(t, folder+"/"+filename)
	return string(b)
}
