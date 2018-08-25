package main

import (
	"reflect"
	"testing"

	"github.com/opendevstack/tailor/openshift"
)

func TestGetFilter(t *testing.T) {
	actual, err := getFilter("pvc", "")
	expected := &openshift.ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilter("pvc,dc", "")
	expected = &openshift.ResourceFilter{
		Kinds: []string{"DeploymentConfig", "PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilter("pvc,persistentvolumeclaim,PersistentVolumeClaim", "")
	expected = &openshift.ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilter("pvb", "")
	expected = nil
	if err == nil {
		t.Errorf("Expected to detect unknown kind pvb.")
	}

	actual, err = getFilter("dc/foo", "")
	expected = &openshift.ResourceFilter{
		Kinds: []string{},
		Name:  "DeploymentConfig/foo",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilter("pvc", "name=foo")
	expected = &openshift.ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "name=foo",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilter("pvc,dc", "name=foo")
	expected = &openshift.ResourceFilter{
		Kinds: []string{"DeploymentConfig", "PersistentVolumeClaim"},
		Name:  "",
		Label: "name=foo",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}
}
