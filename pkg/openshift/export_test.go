package openshift

import (
	"testing"

	"github.com/opendevstack/tailor/internal/test/helper"
)

type mockOcExportClient struct {
	t       *testing.T
	fixture string
}

func (c *mockOcExportClient) Export(target string, label string) ([]byte, error) {
	return helper.ReadFixtureFile(c.t, "export/"+c.fixture), nil
}

func newResourceFilterOrFatal(t *testing.T, kindArg string, selectorFlag string, excludeFlag string) *ResourceFilter {
	filter, err := NewResourceFilter(kindArg, selectorFlag, excludeFlag)
	if err != nil {
		t.Fatal(err)
	}
	return filter
}

func TestExportAsTemplateFile(t *testing.T) {
	tests := map[string]struct {
		fixture         string
		goldenTemplate  string
		filter          *ResourceFilter
		withAnnotations bool
	}{
		"Without annotations": {
			fixture:         "is.yml",
			goldenTemplate:  "is.yml",
			filter:          newResourceFilterOrFatal(t, "is", "", ""),
			withAnnotations: false,
		},
		"With annotations": {
			fixture:         "is.yml",
			goldenTemplate:  "is-annotation.yml",
			filter:          newResourceFilterOrFatal(t, "is", "", ""),
			withAnnotations: true,
		},
		"Works with generateName": {
			fixture:         "rolebinding-generate-name.yml",
			goldenTemplate:  "rolebinding-generate-name.yml",
			filter:          newResourceFilterOrFatal(t, "rolebinding", "", ""),
			withAnnotations: false,
		},
		"Respects filter": {
			fixture:         "is.yml",
			goldenTemplate:  "empty.yml",
			filter:          newResourceFilterOrFatal(t, "bc", "", ""),
			withAnnotations: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &mockOcExportClient{t: t, fixture: tc.fixture}
			actual, err := ExportAsTemplateFile(tc.filter, tc.withAnnotations, c)
			if err != nil {
				t.Fatal(err)
			}

			expected := string(helper.ReadGoldenFile(t, "export/"+tc.goldenTemplate))

			if expected != actual {
				t.Fatalf("Expected template:\n%s\n--- Got template: --- \n%s", expected, actual)
			}
		})
	}
}
