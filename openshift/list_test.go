package openshift

import (
	"testing"
)

func TestConfigFilterByKind(t *testing.T) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    creationTimestamp: null
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    creationTimestamp: null
    name: bar
  data:
    bar: baz
kind: List
metadata: {}
`)

	config := NewConfigFromList(byteList)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}

	list := &ResourceList{Filter: filter}
	list.AppendItems(config)

	if len(list.Items) != 1 {
		t.Errorf("One item should have been extracted, got %v items.", len(list.Items))
		return
	}

	item := list.Items[0]
	if item.Kind != "PersistentVolumeClaim" {
		t.Errorf("Item should have been a PersistentVolumeClaim, got %s.", item.Kind)
	}
}

func TestConfigFilterByName(t *testing.T) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    creationTimestamp: null
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    creationTimestamp: null
    name: bar
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
    storageClassName: gp2
  status: {}
kind: List
metadata: {}
`)

	config := NewConfigFromList(byteList)

	filter := &ResourceFilter{
		Kinds: []string{},
		Name:  "PersistentVolumeClaim/foo",
		Label: "",
	}

	list := &ResourceList{Filter: filter}
	list.AppendItems(config)

	if len(list.Items) != 1 {
		t.Errorf("One item should have been extracted, got %v items.", len(list.Items))
		return
	}

	item := list.Items[0]
	if item.Name != "foo" {
		t.Errorf("Item should have had name foo, got %s.", item.Name)
	}
}

func TestConfigFilterBySelector(t *testing.T) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    creationTimestamp: null
    labels:
      app: foo
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    creationTimestamp: null
    labels:
      app: bar
    name: bar
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
    storageClassName: gp2
  status: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    creationTimestamp: null
    labels:
      app: foo
    name: foo
  data:
    bar: baz
- apiVersion: v1
  kind: ConfigMap
  metadata:
    creationTimestamp: null
    labels:
      app: bar
    name: bar
  data:
    bar: baz
- apiVersion: v1
  data:
    auth-token: abcdef
  kind: Secret
  metadata:
    creationTimestamp: null
    name: bar
    labels:
      app: bar
  type: opaque
kind: List
metadata: {}
`)

	config := NewConfigFromList(byteList)

	pvcFilter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "app=foo",
	}
	cmFilter := &ResourceFilter{
		Kinds: []string{"ConfigMap"},
		Name:  "",
		Label: "app=foo",
	}
	secretFilter := &ResourceFilter{
		Kinds: []string{"Secret"},
		Name:  "",
		Label: "app=foo",
	}

	pvcList := &ResourceList{Filter: pvcFilter}
	pvcList.AppendItems(config)

	if len(pvcList.Items) != 1 {
		t.Errorf("One item should have been extracted, got %v items.", len(pvcList.Items))
	}

	_, err := pvcList.GetItem("PersistentVolumeClaim", "foo")
	if err != nil {
		t.Errorf("Item foo should have been present.")
	}

	cmList := &ResourceList{Filter: cmFilter}
	cmList.AppendItems(config)

	if len(cmList.Items) != 1 {
		t.Errorf("One item should have been extracted, got %v items.", len(cmList.Items))
	}

	_, err = cmList.GetItem("ConfigMap", "foo")
	if err != nil {
		t.Errorf("Item should have been present.")
	}

	secretList := &ResourceList{Filter: secretFilter}
	secretList.AppendItems(config)

	if len(secretList.Items) != 0 {
		t.Errorf("No item should have been extracted, got %v items.", len(secretList.Items))
	}
}
