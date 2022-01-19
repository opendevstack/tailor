package openshift

import (
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
)

func TestNewResourceFilter(t *testing.T) {
	actual, err := NewResourceFilter("pvc", "", []string{})
	expected := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvc,dc", "", []string{})
	expected = &ResourceFilter{
		Kinds: []string{"DeploymentConfig", "PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvc,persistentvolumeclaim,PersistentVolumeClaim", "", []string{})
	expected = &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	_, err = NewResourceFilter("pvb", "", []string{})
	if err == nil {
		t.Errorf("Expected to detect unknown kind pvb.")
	}

	actual, err = NewResourceFilter("dc/foo", "", []string{})
	expected = &ResourceFilter{
		Kinds: []string{},
		Name:  "DeploymentConfig/foo",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvc", "name=foo", []string{})
	expected = &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "name=foo",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("pvc,dc", "name=foo", []string{})
	expected = &ResourceFilter{
		Kinds: []string{"DeploymentConfig", "PersistentVolumeClaim"},
		Name:  "",
		Label: "name=foo",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("hpa", "", []string{})
	expected = &ResourceFilter{
		Kinds: []string{"HorizontalPodAutoscaler"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("statefulset", "", []string{})
	expected = &ResourceFilter{
		Kinds: []string{"StatefulSet"},
		Name:  "",
		Label: "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

	actual, err = NewResourceFilter("", "", []string{"rolebinding", "serviceaccount"})
	expected = &ResourceFilter{
		ExcludedKinds: []string{"RoleBinding", "ServiceAccount"},
		Kinds:         []string{},
		Name:          "",
		Label:         "",
	}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Errorf("Kinds incorrect, got: %v, want: %v.", actual, expected)
	}

}

func TestConvertToKinds(t *testing.T) {
	tests := map[string]struct {
		filter *ResourceFilter
		want   string
	}{
		"kinds": {
			filter: &ResourceFilter{
				ExcludedKinds: []string{},
				Kinds:         []string{"RoleBinding", "ServiceAccount"},
				Name:          "",
				Label:         "",
			},
			want: "RoleBinding,ServiceAccount",
		},
		"excluded kinds": {
			filter: &ResourceFilter{
				ExcludedKinds: []string{"RoleBinding", "ServiceAccount"},
				Kinds:         []string{},
				Name:          "",
				Label:         "",
			},
			want: "Service,Route,DeploymentConfig,Deployment,BuildConfig,ImageStream,PersistentVolumeClaim,Template,ConfigMap,Secret,CronJob,Job,LimitRange,ResourceQuota,HorizontalPodAutoscaler,StatefulSet",
		},
		"kinds and excluded kinds": {
			filter: &ResourceFilter{
				ExcludedKinds: []string{"RoleBinding", "Route"},
				Kinds:         []string{"Service", "Route", "DeploymentConfig"},
				Name:          "",
				Label:         "",
			},
			want: "Service,DeploymentConfig",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.filter.ConvertToKinds()
			if got != tc.want {
				t.Errorf("Got: %+v, want: %+v. Filter is: %+v", got, tc.want, tc.filter)
			}
		})
	}
}

func TestSatisfiedBy(t *testing.T) {
	bc := []byte(
		`kind: BuildConfig
metadata:
  labels:
    app: foo
  name: foo`)
	tests := map[string]struct {
		kindArg      string
		selectorFlag string
		excludes     []string
		config       []byte
		expected     bool
	}{
		"item is included when no constraints are specified": {
			kindArg:      "",
			selectorFlag: "",
			excludes:     []string{},
			config:       bc,
			expected:     true,
		},
		"item is included when kind is specified": {
			kindArg:      "bc",
			selectorFlag: "",
			excludes:     []string{},
			config:       bc,
			expected:     true,
		},
		"item is included when name is specified": {
			kindArg:      "bc/foo",
			selectorFlag: "",
			excludes:     []string{},
			config:       bc,
			expected:     true,
		},
		"item is included when label is specified": {
			kindArg:      "",
			selectorFlag: "app=foo",
			excludes:     []string{},
			config:       bc,
			expected:     true,
		},
		"item is excluded when only some other kind is specified": {
			kindArg:      "is",
			selectorFlag: "",
			excludes:     []string{},
			config:       bc,
			expected:     false,
		},
		"item is excluded when kind is excluded": {
			kindArg:      "",
			selectorFlag: "",
			excludes:     []string{"bc"},
			config:       bc,
			expected:     false,
		},
		"item is excluded when name is excluded": {
			kindArg:      "",
			selectorFlag: "",
			excludes:     []string{"bc/foo"},
			config:       bc,
			expected:     false,
		},
		"item is excluded when label is excluded": {
			kindArg:      "",
			selectorFlag: "",
			excludes:     []string{"app=foo"},
			config:       bc,
			expected:     false,
		},
		"item is excluded when multiple excludes are given that match": {
			kindArg:      "",
			selectorFlag: "",
			excludes:     []string{"app=foo", "bc/foo"},
			config:       bc,
			expected:     false,
		},
		"item is excluded when multiple excludes are given that partially match": {
			kindArg:      "",
			selectorFlag: "",
			excludes:     []string{"app=foobar", "bc/foo"},
			config:       bc,
			expected:     false,
		},
		"item is not excluded when multiple excludes are given that do not match": {
			kindArg:      "",
			selectorFlag: "",
			excludes:     []string{"app=foobar", "dc/foo"},
			config:       bc,
			expected:     true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			item, err := makeItem(tc.config)
			if err != nil {
				t.Fatal(err)
			}
			filter, err := NewResourceFilter(tc.kindArg, tc.selectorFlag, tc.excludes)
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
	err := yaml.Unmarshal(config, &f)
	if err != nil {
		return nil, err
	}
	m := f.(map[string]interface{})
	return NewResourceItem(m, "template")
}
