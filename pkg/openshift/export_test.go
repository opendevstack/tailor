package openshift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opendevstack/tailor/internal/test/helper"
)

type mockOcExportClient struct {
	t       *testing.T
	fixture string
}

func (c *mockOcExportClient) Export(target string, label string) ([]byte, error) {
	return helper.ReadFixtureFile(c.t, "export/"+c.fixture), nil
}

func newResourceFilterOrFatal(t *testing.T, kindArg string, selectorFlag string, excludes []string) *ResourceFilter {
	filter, err := NewResourceFilter(kindArg, selectorFlag, excludes)
	if err != nil {
		t.Fatal(err)
	}
	return filter
}

func TestExportAsTemplateFile(t *testing.T) {
	tests := map[string]struct {
		fixture                string
		goldenTemplate         string
		filter                 *ResourceFilter
		withAnnotations        bool
		trimAnnotations        []string
		namespace              string
		withHardcodedNamespace bool
	}{
		"Without all annotations": {
			fixture:                "is.yml",
			goldenTemplate:         "is.yml",
			filter:                 newResourceFilterOrFatal(t, "is", "", []string{}),
			withAnnotations:        false,
			trimAnnotations:        []string{},
			namespace:              "foo",
			withHardcodedNamespace: true,
		},
		"With all annotations": {
			fixture:                "is.yml",
			goldenTemplate:         "is-annotation.yml",
			filter:                 newResourceFilterOrFatal(t, "is", "", []string{}),
			withAnnotations:        true,
			trimAnnotations:        []string{},
			namespace:              "foo",
			withHardcodedNamespace: true,
		},
		"With trimmed annotation": {
			fixture:                "is.yml",
			goldenTemplate:         "is-trimmed-annotation.yml",
			filter:                 newResourceFilterOrFatal(t, "is", "", []string{}),
			withAnnotations:        false,
			trimAnnotations:        []string{"description"},
			namespace:              "foo",
			withHardcodedNamespace: true,
		},
		"With trimmed annotation prefix": {
			fixture:                "is.yml",
			goldenTemplate:         "is-trimmed-annotation-prefix.yml",
			filter:                 newResourceFilterOrFatal(t, "is", "", []string{}),
			withAnnotations:        false,
			trimAnnotations:        []string{"openshift.io/"},
			namespace:              "foo",
			withHardcodedNamespace: true,
		},
		"With TAILOR_NAMESPACE": {
			fixture:                "bc.yml",
			goldenTemplate:         "bc.yml",
			filter:                 newResourceFilterOrFatal(t, "bc", "", []string{}),
			withAnnotations:        false,
			trimAnnotations:        []string{},
			namespace:              "foo-dev",
			withHardcodedNamespace: false,
		},
		"Works with generateName": {
			fixture:                "rolebinding-generate-name.yml",
			goldenTemplate:         "rolebinding-generate-name.yml",
			filter:                 newResourceFilterOrFatal(t, "rolebinding", "", []string{}),
			withAnnotations:        false,
			trimAnnotations:        []string{},
			namespace:              "foo",
			withHardcodedNamespace: true,
		},
		"Respects filter": {
			fixture:                "is.yml",
			goldenTemplate:         "empty.yml",
			filter:                 newResourceFilterOrFatal(t, "bc", "", []string{}),
			withAnnotations:        false,
			namespace:              "foo",
			withHardcodedNamespace: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &mockOcExportClient{t: t, fixture: tc.fixture}
			actual, err := ExportAsTemplateFile(tc.filter, tc.withAnnotations, tc.namespace, tc.withHardcodedNamespace, tc.trimAnnotations, c)
			if err != nil {
				t.Fatal(err)
			}

			expected := string(helper.ReadGoldenFile(t, "export/"+tc.goldenTemplate))

			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Fatalf("Expected template mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
