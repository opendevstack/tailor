package openshift

import (
	"testing"
)

func TestConfigNoop(t *testing.T) {

	localProcessedTemplate := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    annotations:
      volume.beta.kubernetes.io/storage-provisioner: kubernetes.io/aws-ebs
    creationTimestamp: null
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

	remoteExport := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    annotations:
      pv.kubernetes.io/bind-completed: "yes"
      pv.kubernetes.io/bound-by-controller: "yes"
      volume.beta.kubernetes.io/storage-provisioner: kubernetes.io/aws-ebs
    creationTimestamp: null
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

	remoteConfig := NewConfigFromList(remoteExport)
	localConfig := NewConfigFromList(localProcessedTemplate)
	remoteResourceList := NewResourceList("PersistentVolumeClaim", remoteConfig)
	localResourceList := NewResourceList("PersistentVolumeClaim", localConfig)
	changeset := NewChangeset(remoteResourceList, localResourceList)

	if !changeset.Blank() {
		t.Errorf("Changeset is not blank")
	}
}

func TestConfigUpdate(t *testing.T) {

	localProcessedTemplate := []byte(
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
kind: List
metadata: {}
`)

	remoteExport := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    creationTimestamp: null
    name: foo
    annotations:
    kubectl.kubernetes.io/last-applied-configuration: >
      {"apiVersion":"1"}
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

	remoteConfig := NewConfigFromList(remoteExport)
	localConfig := NewConfigFromList(localProcessedTemplate)
	remoteResourceList := NewResourceList("PersistentVolumeClaim", remoteConfig)
	localResourceList := NewResourceList("PersistentVolumeClaim", localConfig)
	changeset := NewChangeset(remoteResourceList, localResourceList)

	if len(changeset.Update) != 1 {
		t.Errorf("Changeset.Update is blank but should not be")
	}
}

func TestConfigCreation(t *testing.T) {

	localProcessedTemplate := []byte(
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
kind: List
metadata: {}
`)

	remoteExport := []byte(
		`apiVersion: v1
items:
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
        storage: 5Gi
    storageClassName: gp2
  status: {}
kind: List
metadata: {}
`)

	remoteConfig := NewConfigFromList(remoteExport)
	localConfig := NewConfigFromList(localProcessedTemplate)
	remoteResourceList := NewResourceList("PersistentVolumeClaim", remoteConfig)
	localResourceList := NewResourceList("PersistentVolumeClaim", localConfig)
	changeset := NewChangeset(remoteResourceList, localResourceList)

	if len(changeset.Create) != 1 {
		t.Errorf("Changeset.Create is blank but should not be")
	}
}

func TestConfigDeletion(t *testing.T) {

	localProcessedTemplate := []byte{}

	remoteExport := []byte(
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
kind: List
metadata: {}
`)

	remoteConfig := NewConfigFromList(remoteExport)
	localConfig := NewConfigFromList(localProcessedTemplate)
	remoteResourceList := NewResourceList("PersistentVolumeClaim", remoteConfig)
	localResourceList := NewResourceList("PersistentVolumeClaim", localConfig)
	changeset := NewChangeset(remoteResourceList, localResourceList)

	if len(changeset.Delete) != 1 {
		t.Errorf("Changeset.Delete is blank but should not be")
	}
}
