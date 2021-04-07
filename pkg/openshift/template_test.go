package openshift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opendevstack/tailor/internal/test/helper"
	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/utils"
)

func TestTemplateContainsTailorNamespaceParam(t *testing.T) {
	tests := map[string]struct {
		filename     string
		wantContains bool
		wantError    string
	}{
		"contains param": {
			filename:     "with-tailor-namespace-param.yml",
			wantContains: true,
			wantError:    "",
		},
		"without param": {
			filename:     "without-tailor-namespace-param.yml",
			wantContains: false,
			wantError:    "",
		},
		"invalid template": {
			filename:     "invalid-template.yml",
			wantContains: false,
			wantError:    "Not a valid template. Did you forget to add the template header?\n\napiVersion: v1\nkind: Template\nobjects: [...]",
		},
		"template with blank parameters": {
			filename:     "template-blank-parameters.yml",
			wantContains: false,
			wantError:    "",
		},
		"garbage": {
			filename:     "garbage.yml",
			wantContains: false,
			wantError:    "Not a valid template. Please see https://github.com/opendevstack/tailor#template-authoring",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			contains, err := templateContainsTailorNamespaceParam(
				"../../internal/test/fixtures/template-param-detection/" + tc.filename,
			)
			if len(tc.wantError) == 0 {
				if err != nil {
					t.Fatalf("Could not determine if the template contains the param: %s", err)
				}
			} else {
				if err == nil {
					t.Fatalf("Want error '%s', but no error occured", tc.wantError)
				}
				if tc.wantError != err.Error() {
					t.Fatalf("Want error '%s', got '%s'", tc.wantError, err)
				}
			}
			if tc.wantContains != contains {
				t.Fatalf("Want template containing param '%t', got '%t'", tc.wantContains, contains)
			}
		})
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
