package openshift

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestDiff(t *testing.T) {

	diffs := map[string]struct {
		currentAnnotations []byte
		currentData        []byte
		desiredAnnotations []byte
		desiredData        []byte
		expectedDiff       string
		expectedPatches    []*jsonPatch
	}{
		"Modifying a data field": {
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
			[]*jsonPatch{
				&jsonPatch{
					Op:    "replace",
					Path:  "/data/foo",
					Value: "baz",
				},
			},
		},
		"Adding a data field": {
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
			[]*jsonPatch{
				&jsonPatch{
					Op:    "add",
					Path:  "/data/baz",
					Value: "qux",
				},
			},
		},
		"Removing a data field": {
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
			[]*jsonPatch{
				&jsonPatch{
					Op:   "remove",
					Path: "/data/foo",
				},
			},
		},
		"Adding an annotation": {
			[]byte("{}"),
			[]byte("{}"),
			[]byte("{foo: bar}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -2,7 +2,8 @@
 data: {}
 kind: ConfigMap
 metadata:
-  annotations: {}
+  annotations:
+    foo: bar
   labels:
     app: bar
   name: bar
`,
			[]*jsonPatch{
				&jsonPatch{
					Op:    "add",
					Path:  "/metadata/annotations/foo",
					Value: "bar",
				},
				&jsonPatch{
					Op:    "add",
					Path:  "/metadata/annotations/tailor.opendevstack.org~1managed-annotations",
					Value: "foo",
				},
			},
		},
		"Removing an annotation": {
			[]byte("{foo: bar, tailor.opendevstack.org/managed-annotations: foo}"),
			[]byte("{}"),
			[]byte("{}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -2,8 +2,7 @@
 data: {}
 kind: ConfigMap
 metadata:
-  annotations:
-    foo: bar
+  annotations: {}
   labels:
     app: bar
   name: bar
`,
			[]*jsonPatch{
				&jsonPatch{
					Op:   "remove",
					Path: "/metadata/annotations/foo",
				},
				&jsonPatch{
					Op:   "remove",
					Path: "/metadata/annotations/tailor.opendevstack.org~1managed-annotations",
				},
			},
		},
		"Modifying an annotation": {
			[]byte("{foo: bar, tailor.opendevstack.org/managed-annotations: foo}"),
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
   labels:
     app: bar
   name: bar
`,
			[]*jsonPatch{
				&jsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/foo",
					Value: "baz",
				},
			},
		},
		"Modifying a non-managed annotation": {
			[]byte("{foo: bar, baz: qux, tailor.opendevstack.org/managed-annotations: foo}"),
			[]byte("{}"),
			[]byte("{foo: bar, baz: zab}"),
			[]byte("{}"),
			`--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -3,7 +3,7 @@
 kind: ConfigMap
 metadata:
   annotations:
-    baz: qux
+    baz: zab
     foo: bar
   labels:
     app: bar
`,
			[]*jsonPatch{
				&jsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/baz",
					Value: "zab",
				},
				&jsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/tailor.opendevstack.org~1managed-annotations",
					Value: "baz,foo",
				},
			},
		},
		"Managing a non-managed annotation": {
			[]byte("{foo: bar, baz: qux, tailor.opendevstack.org/managed-annotations: foo}"),
			[]byte("{}"),
			[]byte("{foo: bar, baz: qux}"),
			[]byte("{}"),
			`Only annotations used by Tailor internally differ. Use --diff=json to see details.
`,
			[]*jsonPatch{
				&jsonPatch{
					Op:    "replace",
					Path:  "/metadata/annotations/tailor.opendevstack.org~1managed-annotations",
					Value: "baz,foo",
				},
			},
		},
	}

	for name, tt := range diffs {
		t.Run(name, func(t *testing.T) {
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
			changes, err := calculateChanges(desiredItem, currentItem, []string{}, true)
			if err != nil {
				t.Fatal(err)
			}
			change := changes[0]
			actualDiff := change.Diff(true)
			if actualDiff != tt.expectedDiff {
				t.Fatalf(
					"Diff()\n===== expected =====\n%s\n===== actual =====\n%s",
					tt.expectedDiff,
					actualDiff,
				)
			}
			actualPatches := change.Patches
			if !reflect.DeepEqual(actualPatches, tt.expectedPatches) {
				t.Fatalf(
					"Patches()\n===== expected =====\n%s\n===== actual =====\n%s",
					pretty(tt.expectedPatches),
					pretty(actualPatches),
				)
			}
		})
	}
}

func pretty(jp []*jsonPatch) string {
	var b []byte
	b, _ = json.MarshalIndent(jp, "", "  ")
	return string(b)
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
