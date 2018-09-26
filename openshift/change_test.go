package openshift

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDiff(t *testing.T) {

	diffs := []struct {
		currentAnnotations []byte
		currentData        []byte
		desiredAnnotations []byte
		desiredData        []byte
		expectedDiff       string
		expectedPatches    []*JsonPatch
	}{
		{ // Modifying a data field
			[]byte("{}"),
			[]byte("{foo: bar}"),
			[]byte("{}"),
			[]byte("{foo: baz}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -1,6 +1,6 @@
 apiVersion: v1
 data:
-  foo: bar
+  foo: baz
 kind: ConfigMap
 metadata:
   annotations: {}
`,
			[]*JsonPatch{
				&JsonPatch{
					Op:    "replace",
					Path:  "/data/foo",
					Value: "baz",
				},
			},
		},
		{ // Adding a data field
			[]byte("{}"),
			[]byte("{foo: bar}"),
			[]byte("{}"),
			[]byte("{foo: bar, baz: qux}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -1,5 +1,6 @@
 apiVersion: v1
 data:
+  baz: qux
   foo: bar
 kind: ConfigMap
 metadata:
`,
			[]*JsonPatch{
				&JsonPatch{
					Op:    "add",
					Path:  "/data/baz",
					Value: "qux",
				},
			},
		},
		{ // Removing a data field
			[]byte("{}"),
			[]byte("{foo: bar}"),
			[]byte("{}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -1,6 +1,5 @@
 apiVersion: v1
-data:
-  foo: bar
+data: {}
 kind: ConfigMap
 metadata:
   annotations: {}
`,
			[]*JsonPatch{
				&JsonPatch{
					Op:   "remove",
					Path: "/data/foo",
				},
			},
		},
		{ // Adding an annotation
			[]byte("{}"),
			[]byte("{}"),
			[]byte("{foo: bar}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -2,7 +2,9 @@
 data: {}
 kind: ConfigMap
 metadata:
-  annotations: {}
+  annotations:
+    foo: bar
+    managed-annotations.tailor.opendevstack.org: foo
   labels:
     app: bar
   name: bar
`,
			[]*JsonPatch{
				&JsonPatch{
					Op:    "add",
					Path:  "/metadata/annotations/foo",
					Value: "bar",
				},
				&JsonPatch{
					Op:    "add",
					Path:  "/metadata/annotations/managed-annotations.tailor.opendevstack.org",
					Value: "foo",
				},
			},
		},
		{ // Removing an annotation
			[]byte("{foo: bar, managed-annotations.tailor.opendevstack.org: foo}"),
			[]byte("{}"),
			[]byte("{}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -2,9 +2,7 @@
 data: {}
 kind: ConfigMap
 metadata:
-  annotations:
-    foo: bar
-    managed-annotations.tailor.opendevstack.org: foo
+  annotations: {}
   labels:
     app: bar
   name: bar
`,
			[]*JsonPatch{
				&JsonPatch{
					Op:   "remove",
					Path: "/metadata/annotations/foo",
				},
				&JsonPatch{
					Op:   "remove",
					Path: "/metadata/annotations/managed-annotations.tailor.opendevstack.org",
				},
			},
		},
		{ // Modifying an annotation
			[]byte("{foo: bar, managed-annotations.tailor.opendevstack.org: foo}"),
			[]byte("{}"),
			[]byte("{foo: baz}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -3,7 +3,7 @@
 kind: ConfigMap
 metadata:
   annotations:
-    foo: bar
+    foo: baz
     managed-annotations.tailor.opendevstack.org: foo
   labels:
     app: bar
`,
			[]*JsonPatch{
				&JsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/foo",
					Value: "baz",
				},
			},
		},
		{ // Modifying a non-managed annotation
			[]byte("{foo: bar, baz: qux, managed-annotations.tailor.opendevstack.org: foo}"),
			[]byte("{}"),
			[]byte("{foo: bar, baz: zab}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -3,9 +3,9 @@
 kind: ConfigMap
 metadata:
   annotations:
-    baz: qux
+    baz: zab
     foo: bar
-    managed-annotations.tailor.opendevstack.org: foo
+    managed-annotations.tailor.opendevstack.org: baz,foo
   labels:
     app: bar
   name: bar
`,
			[]*JsonPatch{
				&JsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/baz",
					Value: "zab",
				},
				&JsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/managed-annotations.tailor.opendevstack.org",
					Value: "baz,foo",
				},
			},
		},
		{ // Managing a non-managed annotation
			[]byte("{foo: bar, baz: qux, managed-annotations.tailor.opendevstack.org: foo}"),
			[]byte("{}"),
			[]byte("{foo: bar, baz: qux}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -5,7 +5,7 @@
   annotations:
     baz: qux
     foo: bar
-    managed-annotations.tailor.opendevstack.org: foo
+    managed-annotations.tailor.opendevstack.org: baz,foo
   labels:
     app: bar
   name: bar
`,
			[]*JsonPatch{
				&JsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/managed-annotations.tailor.opendevstack.org",
					Value: "baz,foo",
				},
			},
		},
	}

	for _, tt := range diffs {
		currentItem := getItem(
			t,
			getConfigMapForDiff(tt.currentAnnotations, tt.currentData),
			"platform",
		)
		desiredItem := getItem(
			t,
			getConfigMapForDiff(tt.desiredAnnotations, tt.desiredData),
			"template",
		)
		changes, err := desiredItem.ChangesFrom(currentItem, []string{})
		if err != nil {
			t.Errorf(err.Error())
		}
		change := changes[0]
		actualDiff := change.Diff()
		if actualDiff != tt.expectedDiff {
			t.Errorf(
				"Diff()\n===== expected =====\n%s\n===== actual =====\n%s",
				tt.expectedDiff,
				actualDiff,
			)
		}
		actualPatches := change.Patches
		if !reflect.DeepEqual(actualPatches, tt.expectedPatches) {
			t.Errorf(
				"Diff()\n===== expected =====\n%s\n===== actual =====\n%s",
				tt.expectedPatches,
				actualPatches,
			)
		}
	}
}

func getConfigMapForDiff(annotations, data []byte) []byte {
	config := []byte(
		`apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app: bar
  annotations: ANNOTATIONS
  name: bar
data: DATA`)
	config = bytes.Replace(config, []byte("ANNOTATIONS"), annotations, -1)
	return bytes.Replace(config, []byte("DATA"), data, -1)
}
