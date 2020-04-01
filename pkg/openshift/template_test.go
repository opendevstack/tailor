package openshift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opendevstack/tailor/internal/test/helper"
	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/utils"
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

func TestCalculateParamFiles(t *testing.T) {
	tests := map[string]struct {
		namespace     string
		templateName  string
		paramDir      string
		paramFileFlag []string
		fs            utils.FileStater
		expected      []string
	}{
		"template is foo.yml and corresponding param file exists": {
			namespace:     "foo",
			templateName:  "bar.yml",
			paramDir:      ".", // default
			paramFileFlag: []string{},
			fs:            &helper.SomeFilesExistFS{Existing: []string{"bar.env", "foo.env"}},
			expected:      []string{"bar.env", "foo.env"},
		},
		"template is bar.yml but corresponding param file does not exist": {
			namespace:     "foo",
			templateName:  "bar.yml",
			paramDir:      ".", // default
			paramFileFlag: []string{},
			fs:            &helper.SomeFilesExistFS{Existing: []string{"foo.env"}},
			expected:      []string{"foo.env"},
		},
		"template is bar.yml and no files exist": {
			namespace:     "foo",
			templateName:  "bar.yml",
			paramDir:      ".", // default
			paramFileFlag: []string{},
			fs:            &helper.SomeFilesExistFS{},
			expected:      []string{},
		},
		"template is foo.yml and corresponding param file exists in param dir": {
			namespace:     "foo",
			templateName:  "bar.yml",
			paramDir:      "foo", // default
			paramFileFlag: []string{},
			fs:            &helper.SomeFilesExistFS{Existing: []string{"foo/bar.env", "foo.env"}},
			expected:      []string{"foo/bar.env", "foo.env"},
		},
		"template is foo.yml but corresponding param file does not exist in param dir": {
			namespace:     "foo",
			templateName:  "bar.yml",
			paramDir:      "foo", // default
			paramFileFlag: []string{},
			fs:            &helper.SomeFilesExistFS{Existing: []string{"foo", "foo.env"}},
			expected:      []string{"foo.env"},
		},
		"param env file is given explicitly": {
			namespace:     "foo",
			templateName:  "bar.yml",
			paramDir:      ".", // default
			paramFileFlag: []string{"foo.env"},
			fs:            &helper.SomeFilesExistFS{Existing: []string{"foo.env"}},
			expected:      []string{"foo.env"},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			globalOptions := cli.InitGlobalOptions(tc.fs)
			compareOptions := &cli.CompareOptions{
				GlobalOptions:    globalOptions,
				NamespaceOptions: &cli.NamespaceOptions{Namespace: tc.namespace},
				ParamFiles:       tc.paramFileFlag,
			}

			actual := calculateParamFiles(tc.templateName, tc.paramDir, compareOptions)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Fatalf("Desired state mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadParamFileBytes(t *testing.T) {
	tests := map[string]struct {
		paramFiles []string
		expected   string
	}{
		"multiple files get concatenated": {
			paramFiles: []string{"foo.env", "bar.env"},
			expected:   "FOO=foo\nBAR=bar\n",
		},
		"missing EOL is handled in concatenation": {
			paramFiles: []string{"baz-without-eol.env", "bar.env"},
			expected:   "BAZ=baz\nBAR=bar\n",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actualParamFiles := []string{}
			for _, f := range tc.paramFiles {
				actualParamFiles = append(actualParamFiles, "../../internal/test/fixtures/param-files/"+f)
			}
			b, err := readParamFileBytes(actualParamFiles, "", "")
			if err != nil {
				t.Fatal(err)
			}
			got := string(b)
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("Result is not expected (-want +got):\n%s", diff)
			}
		})
	}
}
