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

func TestExportAsTemplateFile(t *testing.T) {
	tests := map[string]struct {
		fixture         string
		goldenTemplate  string
		withAnnotations bool
	}{
		"Without annotations": {
			fixture:         "is.yml",
			goldenTemplate:  "is.yml",
			withAnnotations: false,
		},
		"With annotations": {
			fixture:         "is.yml",
			goldenTemplate:  "is-annotation.yml",
			withAnnotations: true,
		},
		"With generateName": {
			fixture:         "rolebinding-generate-name.yml",
			goldenTemplate:  "rolebinding-generate-name.yml",
			withAnnotations: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filter, err := NewResourceFilter("is", "", "")
			if err != nil {
				t.Fatal(err)
			}

			c := &mockOcExportClient{t: t, fixture: tc.fixture}
			actual, err := ExportAsTemplateFile(filter, tc.withAnnotations, c)
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

func TestTemplateContainsTailorNamespaceParam(t *testing.T) {
	contains, err := templateContainsTailorNamespaceParam("../../internal/test/fixtures/template-with-tailor-namespace-param.yml")
	if err != nil {
		t.Errorf("Could not determine if the template contains the param: %s", err)
	}
	if !contains {
		t.Error("Template contains param, but it was not detected")
	}
}
