package main

import (
	"testing"
	"reflect"
	"github.com/michaelsauter/ocdiff/openshift"
)

func TestGetFilters(t *testing.T) {
	actual, err := getFilters("pvc", "")
	expected := map[string]*openshift.ResourceFilter{
		"PersistentVolumeClaim": &openshift.ResourceFilter{
			Kind: "PersistentVolumeClaim",
			Names: []string{},
			Label: "",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilters("pvc,dc", "")
	expected = map[string]*openshift.ResourceFilter{
		"PersistentVolumeClaim": &openshift.ResourceFilter{
			Kind: "PersistentVolumeClaim",
			Names: []string{},
			Label: "",
		},
		"DeploymentConfig": &openshift.ResourceFilter{
			Kind: "DeploymentConfig",
			Names: []string{},
			Label: "",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilters("pvc,persistentvolumeclaim,PersistentVolumeClaim", "")
	expected = map[string]*openshift.ResourceFilter{
		"PersistentVolumeClaim": &openshift.ResourceFilter{
			Kind: "PersistentVolumeClaim",
			Names: []string{},
			Label: "",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilters("pvb", "")
	expected = map[string]*openshift.ResourceFilter{}
	if err == nil {
		t.Errorf("Expected to detect unknown kind pvb.")
	}

	actual, err = getFilters("dc/foo", "")
	expected = map[string]*openshift.ResourceFilter{
		"DeploymentConfig": &openshift.ResourceFilter{
			Kind: "DeploymentConfig",
			Names: []string{"foo"},
			Label: "",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilters("dc/foo,dc/bar", "")
	expected = map[string]*openshift.ResourceFilter{
		"DeploymentConfig": &openshift.ResourceFilter{
			Kind: "DeploymentConfig",
			Names: []string{"foo", "bar"},
			Label: "",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilters("dc/foo,bc/bar", "")
	expected = map[string]*openshift.ResourceFilter{
		"DeploymentConfig": &openshift.ResourceFilter{
			Kind: "DeploymentConfig",
			Names: []string{"foo"},
			Label: "",
		},
		"BuildConfig": &openshift.ResourceFilter{
			Kind: "BuildConfig",
			Names: []string{"bar"},
			Label: "",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilters("pvc", "name=foo")
	expected = map[string]*openshift.ResourceFilter{
		"PersistentVolumeClaim": &openshift.ResourceFilter{
			Kind: "PersistentVolumeClaim",
			Names: []string{},
			Label: "name=foo",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = getFilters("pvc,dc/foobar", "name=foo")
	expected = map[string]*openshift.ResourceFilter{
		"PersistentVolumeClaim": &openshift.ResourceFilter{
			Kind: "PersistentVolumeClaim",
			Names: []string{},
			Label: "name=foo",
		},
		"DeploymentConfig": &openshift.ResourceFilter{
			Kind: "DeploymentConfig",
			Names: []string{"foobar"},
			Label: "name=foo",
		},
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}
}
