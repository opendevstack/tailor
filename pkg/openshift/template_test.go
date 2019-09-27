package openshift

import (
	"testing"

	"github.com/opendevstack/tailor/internal/test/helper"
)

type fakeOcClient struct{}

func (c *fakeOcClient) Export(target string, label string) ([]byte, error) {
	return helper.ReadFixtureFileOrErr("export/is.yml")
}

func TestExportAsTemplateFile(t *testing.T) {
	tests := map[string]struct {
		goldenTemplateFile string
		withAnnotations    bool
	}{
		"Without annotations": {
			goldenTemplateFile: "is.yml",
			withAnnotations:    false,
		},
		"With annotations": {
			goldenTemplateFile: "is-annotation.yml",
			withAnnotations:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filter, err := NewResourceFilter("is", "", "")
			if err != nil {
				t.Fatal(err)
			}

			actual, err := ExportAsTemplateFile(filter, tc.withAnnotations, &fakeOcClient{})
			if err != nil {
				t.Fatal(err)
			}

			expected := string(helper.ReadGoldenFile(t, "export/"+tc.goldenTemplateFile))

			if expected != actual {
				t.Fatalf("Expected template:\n%s\nGot template:\n%s", expected, actual)
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
