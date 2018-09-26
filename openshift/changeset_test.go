package openshift

import (
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
		`apiVersion: v1
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
  status: {}
kind: List
metadata: {}
`)

	platformInput := []byte(
		`apiVersion: v1
items:
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
  status: {}
kind: List
metadata: {}
`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
	}
	platformBasedList := &ResourceList{Filter: filter}
	platformBasedList.CollectItemsFromPlatformList(platformInput)
	templateBasedList := &ResourceList{Filter: filter}
	templateBasedList.CollectItemsFromTemplateList(templateInput)
	changeset, err := NewChangeset(platformBasedList, templateBasedList, false, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if !changeset.Blank() {
		t.Errorf("Changeset is not blank, got %v", changeset.Update[0].JsonPatches(true))
	}
}

func TestConfigUpdate(t *testing.T) {

	templateInput := []byte(
		`apiVersion: v1
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
  status: {}
kind: List
metadata: {}
`)

	platformInput := []byte(
		`apiVersion: v1
items:
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
  status: {}
kind: List
metadata: {}
`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
	}
	platformBasedList := &ResourceList{Filter: filter}
	platformBasedList.CollectItemsFromPlatformList(platformInput)
	templateBasedList := &ResourceList{Filter: filter}
	templateBasedList.CollectItemsFromTemplateList(templateInput)
	changeset, err := NewChangeset(platformBasedList, templateBasedList, false, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changeset.Update) != 1 {
		t.Errorf("Changeset.Update has %d items instead of 1", len(changeset.Update))
	}
}

func TestConfigCreation(t *testing.T) {
	templateInput := []byte(
		`apiVersion: v1
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
  status: {}
kind: List
metadata: {}`)

	platformInput := []byte(
		`apiVersion: v1
items:
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
  status: {}
kind: List
metadata: {}`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
	}
	platformBasedList := &ResourceList{Filter: filter}
	platformBasedList.CollectItemsFromPlatformList(platformInput)
	templateBasedList := &ResourceList{Filter: filter}
	templateBasedList.CollectItemsFromTemplateList(templateInput)
	changeset, err := NewChangeset(platformBasedList, templateBasedList, false, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changeset.Create) != 1 {
		t.Errorf("Changeset.Create is blank but should not be")
	}
}

func TestConfigDeletion(t *testing.T) {

	templateInput := []byte{}

	platformInput := []byte(
		`apiVersion: v1
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
  status: {}
kind: List
metadata: {}
`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
	}
	platformBasedList := &ResourceList{Filter: filter}
	platformBasedList.CollectItemsFromPlatformList(platformInput)
	templateBasedList := &ResourceList{Filter: filter}
	templateBasedList.CollectItemsFromTemplateList(templateInput)
	changeset, err := NewChangeset(platformBasedList, templateBasedList, false, []string{})
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(changeset.Delete) != 1 {
		t.Errorf("Changeset.Delete is blank but should not be")
	}
}
