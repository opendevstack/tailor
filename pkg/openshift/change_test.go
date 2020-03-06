package openshift

import (
	"bytes"
	"testing"
)

func TestDiff(t *testing.T) {

	diffs := map[string]struct {
		currentAnnotations []byte
		currentData        []byte
		desiredAnnotations []byte
		desiredData        []byte
		expectedDiff       string
	}{
		"Modifying a data field": {
			currentAnnotations: []byte("{}"),
			currentData:        []byte("{foo: bar}"),
			desiredAnnotations: []byte("{}"),
			desiredData:        []byte("{foo: baz}"),
			expectedDiff: `--- Current State (OpenShift cluster)
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
		},
		"Adding a data field": {
			currentAnnotations: []byte("{}"),
			currentData:        []byte("{foo: bar}"),
			desiredAnnotations: []byte("{}"),
			desiredData:        []byte("{foo: bar, baz: qux}"),
			expectedDiff: `--- Current State (OpenShift cluster)
+++ Desired State (Processed template)
@@ -1,5 +1,6 @@
 apiVersion: v1
 data:
+  baz: qux
   foo: bar
 kind: ConfigMap
 metadata:
`,
		},
		"Removing a data field": {
			currentAnnotations: []byte("{}"),
			currentData:        []byte("{foo: bar}"),
			desiredAnnotations: []byte("{}"),
			desiredData:        []byte("{}"),
			expectedDiff: `--- Current State (OpenShift cluster)
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
		},
		"Adding an annotation": {
			currentAnnotations: []byte("{}"),
			currentData:        []byte("{}"),
			desiredAnnotations: []byte("{foo: bar}"),
			desiredData:        []byte("{}"),
			expectedDiff: `--- Current State (OpenShift cluster)
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
		},
		"Removing an annotation": {
			currentAnnotations: []byte(`{foo: bar, kubectl.kubernetes.io/last-applied-configuration: '{"apiVersion":"v1","kind":"ConfigMap","metadata":{"annotations":{"foo":"bar"}}}'}`),
			currentData:        []byte("{}"),
			desiredAnnotations: []byte("{}"),
			desiredData:        []byte("{}"),
			expectedDiff: `--- Current State (OpenShift cluster)
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
		},
		"Modifying an annotation": {
			currentAnnotations: []byte(`{foo: bar, kubectl.kubernetes.io/last-applied-configuration: '{"apiVersion":"v1","kind":"ConfigMap","metadata":{"annotations":{"foo":"bar"}}}'}`),
			currentData:        []byte("{}"),
			desiredAnnotations: []byte("{foo: baz}"),
			desiredData:        []byte("{}"),
			expectedDiff: `--- Current State (OpenShift cluster)
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
		},
		"Modifying a non-managed annotation": {
			currentAnnotations: []byte(`{foo: bar, baz: qux, kubectl.kubernetes.io/last-applied-configuration: '{"apiVersion":"v1","kind":"ConfigMap","metadata":{"annotations":{"foo":"bar"}}}'}`),
			currentData:        []byte("{}"),
			desiredAnnotations: []byte("{foo: bar, baz: zab}"),
			desiredData:        []byte("{}"),
			expectedDiff: `--- Current State (OpenShift cluster)
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
		})
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
