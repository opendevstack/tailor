package openshift

import (
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
)

func TestNewResourceFilter(t *testing.T) {
	actual, err := NewResourceFilter("pvc", "", "")
	expected := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvc,dc", "", "")
	expected = &ResourceFilter{
		Kinds: []string{"DeploymentConfig", "PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvc,persistentvolumeclaim,PersistentVolumeClaim", "", "")
	expected = &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvb", "", "")
	expected = nil
	if err == nil {
		t.Errorf("Expected to detect unknown kind pvb.")
	}

	actual, err = NewResourceFilter("dc/foo", "", "")
	expected = &ResourceFilter{
		Kinds: []string{},
		Name:  "DeploymentConfig/foo",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvc", "name=foo", "")
	expected = &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "name=foo",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvc,dc", "name=foo", "")
	expected = &ResourceFilter{
		Kinds: []string{"DeploymentConfig", "PersistentVolumeClaim"},
		Name:  "",
		Label: "name=foo",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}
}

func TestSatisfiedBy(t *testing.T) {
	bc := []byte(
		`kind: BuildConfig
metadata:
  labels:
    app: foo
  name: foo`)
	examples := []struct {
		name         string
		kindArg      string
		selectorFlag string
		excludeFlag  string
		config       []byte
		expected     bool
	}{
		{
			"item is included when no constraints are specified",
			"",
			"",
			"",
			bc,
			true,
		},
		{
			"item is included when kind is specified",
			"bc",
			"",
			"",
			bc,
			true,
		},
		{
			"item is included when name is specified",
			"bc/foo",
			"",
			"",
			bc,
			true,
		},
		{
			"item is included when label is specified",
			"",
			"app=foo",
			"",
			bc,
			true,
		},
		{
			"item is excluded when only some other kind is specified",
			"is",
			"",
			"",
			bc,
			false,
		},
		{
			"item is excluded when kind is excluded",
			"",
			"",
			"bc",
			bc,
			false,
		},
		{
			"item is excluded when name is excluded",
			"",
			"",
			"bc/foo",
			bc,
			false,
		},
		{
			"item is excluded when label is excluded",
			"",
			"",
			"app=foo",
			bc,
			false,
		},
	}

	for _, tc := range examples {
		t.Run(tc.name, func(t *testing.T) {
			item, err := makeItem(tc.config)
			if err != nil {
				t.Fatal(err)
			}
			filter, err := NewResourceFilter(tc.kindArg, tc.selectorFlag, tc.excludeFlag)
			if err != nil {
				t.Fatal(err)
			}
			actual := filter.SatisfiedBy(item)
			if actual != tc.expected {
				t.Errorf("Got: %+v, want: %+v. Filter is: %+v", actual, tc.expected, filter)
			}
		})
	}
}

func makeItem(config []byte) (*ResourceItem, error) {
	var f interface{}
	yaml.Unmarshal(config, &f)
	m := f.(map[string]interface{})
	return NewResourceItem(m, "template")
}
