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
			ignoredPaths := []string{}
			cs, err := NewChangeset(
				platformBasedList,
				templateBasedList,
				upsertOnly,
				allowRecreate,
				ignoredPaths,
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
		expectedPatches        jsonPatches
		expectedDiffGoldenFile string
	}{
		"Without annotations": {
			platformFixture: "is-platform",
			templateFixture: "is-template",
			expectedAction:  "Noop",
			expectedPatches: jsonPatches{},
		},
		"Present in template, not in platform": {
			platformFixture: "is-platform",
			templateFixture: "is-template-annotation",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:   "add",
					Path: "/metadata/annotations",
					Value: map[string]string{
						"bar": "baz",
						"tailor.opendevstack.org/managed-annotations": "bar",
					},
				},
			},
			expectedDiffGoldenFile: "present-in-template-not-in-platform",
		},
		"Present in platform, not in template": {
			platformFixture: "is-platform-annotation",
			templateFixture: "is-template",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:   "remove",
					Path: "/metadata/annotations/bar",
				},
				&jsonPatch{
					Op:   "remove",
					Path: "/metadata/annotations/tailor.opendevstack.org~1managed-annotations",
				},
			},
			expectedDiffGoldenFile: "present-in-platform-not-in-template",
		},
		"Present in both": {
			platformFixture: "is-platform-annotation",
			templateFixture: "is-template-annotation",
			expectedAction:  "Noop",
			expectedPatches: jsonPatches{},
		},
		"Present in platform, changed in template": {
			platformFixture: "is-platform-annotation",
			templateFixture: "is-template-annotation-changed",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/bar",
					Value: "qux",
				},
			},
			expectedDiffGoldenFile: "present-in-platform-changed-in-template",
		},
		"Present in platform, different key in template": {
			platformFixture: "is-platform-annotation",
			templateFixture: "is-template-different-annotation",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:   "remove",
					Path: "/metadata/annotations/bar",
				},
				&jsonPatch{
					Op:    "add",
					Path:  "/metadata/annotations/baz",
					Value: "qux",
				},
				&jsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/tailor.opendevstack.org~1managed-annotations",
					Value: "baz",
				},
			},
			expectedDiffGoldenFile: "present-in-platform-different-key-in-template",
		},
		"Unmanaged in platform added to template": {
			platformFixture: "is-platform-unmanaged",
			templateFixture: "is-template-annotation",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:    "add",
					Path:  "/metadata/annotations/tailor.opendevstack.org~1managed-annotations",
					Value: "bar",
				},
			},
			expectedDiffGoldenFile: "unmanaged-in-platform-added-to-template",
		},
		"Unmanaged in platform, none in template": {
			platformFixture: "is-platform-unmanaged",
			templateFixture: "is-template",
			expectedAction:  "Noop",
			expectedPatches: jsonPatches{},
		},
		"Unmanaged in platform, none in template, and other change in template": {
			platformFixture: "is-platform-unmanaged",
			templateFixture: "is-template-other-change",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:    "replace",
					Path:  "/spec/lookupPolicy/local",
					Value: true,
				},
			},
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
			if len(actualChange.Patches) != len(tc.expectedPatches) {
				t.Fatalf("Expected patches:\n%s\n--- got: ---\n%s", pretty(tc.expectedPatches), actualChange.PrettyJSONPatches())
			}
			for i, got := range actualChange.Patches {
				want := tc.expectedPatches[i]
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("Change diff mismatch (-want +got):\n%s", diff)
				}
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
		expectedPatches jsonPatches
	}{
		"Without annotation in platform": {
			platformFixture: "dc-platform",
			templateFixture: "dc-template",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:   "add",
					Path: "/metadata/annotations",
					Value: map[string]string{
						"tailor.opendevstack.org/applied-config": "{\"/spec/template/spec/containers/0/image\":\"bar/foo:latest\"}",
					},
				},
				&jsonPatch{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/image",
					Value: "bar/foo:latest",
				},
			},
		},
		"With annotation in platform": {
			platformFixture: "dc-platform-annotation-other",
			templateFixture: "dc-template",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:    "add",
					Path:  "/metadata/annotations/tailor.opendevstack.org~1applied-config",
					Value: "{\"/spec/template/spec/containers/0/image\":\"bar/foo:latest\"}",
				},
				&jsonPatch{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/image",
					Value: "bar/foo:latest",
				},
			},
		},
		"Present in platform": {
			platformFixture: "dc-platform-annotation-applied",
			templateFixture: "dc-template",
			expectedAction:  "Noop",
			expectedPatches: jsonPatches{},
		},
		"Present in platform, changed in template": {
			platformFixture: "dc-platform-annotation-applied",
			templateFixture: "dc-template-changed",
			expectedAction:  "Update",
			expectedPatches: jsonPatches{
				&jsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/tailor.opendevstack.org~1applied-config",
					Value: "{\"/spec/template/spec/containers/0/image\":\"bar/foo:experiment\"}",
				},
				&jsonPatch{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/image",
					Value: "bar/foo:experiment",
				},
			},
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
				t.Fatalf("Expected change action to be: %s, got: %s. Patches: \n%s", tc.expectedAction, actualChange.Action, actualChange.PrettyJSONPatches())
			}
			if len(actualChange.Patches) != len(tc.expectedPatches) {
				t.Fatalf("Expected patches:\n%s\n--- got: ---\n%s", pretty(tc.expectedPatches), actualChange.PrettyJSONPatches())
			}
			for i, got := range actualChange.Patches {
				want := tc.expectedPatches[i]
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
		t.Fatalf("Expected change action to be: %s, got: %s. Patches: \n%s", expectedAction, actualChange.Action, actualChange.PrettyJSONPatches())
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
		updates := []string{""}
		for _, u := range changeset.Update {
			updates = append(updates, u.PrettyJSONPatches())
		}
		t.Fatalf("Changeset is not blank, got: update=%s", strings.Join(updates, ", "))
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
	changeset := getChangeset(t, filter, platformInput, templateInput, false, true, []string{"bc:/spec/output/to/name", "bc:/spec/output/imageLabels"})
	actualUpdates := len(changeset.Update)
	expectedUpdates := 0
	if actualUpdates != expectedUpdates {
		t.Errorf("Changeset.Update has %d items instead of %d", actualUpdates, expectedUpdates)
		for i, u := range changeset.Update {
			t.Errorf("Patchset Update#%d: %s", i, u.PrettyJSONPatches())
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

func TestCalculateChangesDifferent(t *testing.T) {
	currentItem := getItem(t, getBuildConfig(), "platform")
	desiredItem := getItem(t, getChangedBuildConfig(), "template")
	changes, err := calculateChanges(desiredItem, currentItem, []string{}, true)
	if err != nil {
		t.Errorf(err.Error())
	}
	change := changes[0]
	if len(change.Patches) != 10 {
		t.Errorf("Got %d instead of %d changes: %s", len(change.Patches), 10, change.PrettyJSONPatches())
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

func getChangeset(t *testing.T, filter *ResourceFilter, platformInput, templateInput []byte, upsertOnly bool, allowRecreate bool, ignoredPaths []string) *Changeset {
	platformBasedList, err := NewPlatformBasedResourceList(filter, platformInput)
	if err != nil {
		t.Error("Could not create platform based list:", err)
	}
	templateBasedList, err := NewTemplateBasedResourceList(filter, templateInput)
	if err != nil {
		t.Error("Could not create template based list:", err)
	}
	changeset, err := NewChangeset(platformBasedList, templateBasedList, upsertOnly, allowRecreate, ignoredPaths)
	if err != nil {
		t.Error("Could not create changeset:", err)
	}
	return changeset
}

func getGoldenDiff(t *testing.T, folder string, filename string) string {
	b := helper.ReadGoldenFile(t, folder+"/"+filename)
	return string(b)
}
