package cli

import (
	"testing"

	"github.com/opendevstack/tailor/internal/test/helper"
	"github.com/opendevstack/tailor/pkg/utils"
)

func TestResolvedFile(t *testing.T) {
	tests := map[string]struct {
		fileFlag      string
		namespaceFlag string
		fs            utils.FileStater
		expected      string
	}{
		"no file flag and no namespace flag given": {
			fileFlag:      "Tailorfile", // default
			namespaceFlag: "",
			fs:            &helper.SomeFilesExistFS{Existing: []string{"Tailorfile"}},
			expected:      "Tailorfile",
		},
		"no file flag given but namespace flag given and namespaced file exists": {
			fileFlag:      "Tailorfile", // default
			namespaceFlag: "foo",
			fs:            &helper.SomeFilesExistFS{Existing: []string{"Tailorfile.foo"}},
			expected:      "Tailorfile.foo",
		},
		"no file flag given but namespace flag given and namespaced file does not exist": {
			fileFlag:      "Tailorfile", // default
			namespaceFlag: "foo",
			fs:            &helper.SomeFilesExistFS{},
			expected:      "Tailorfile",
		},
		"file flag given and no namespace flag given": {
			fileFlag:      "mytailorfile",
			namespaceFlag: "",
			fs:            &helper.SomeFilesExistFS{Existing: []string{"mytailorfile"}},
			expected:      "mytailorfile",
		},
		"file flag and namespace flag given": {
			fileFlag:      "mytailorfile",
			namespaceFlag: "foo",
			fs:            &helper.SomeFilesExistFS{Existing: []string{"mytailorfile"}},
			expected:      "mytailorfile",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			o := InitGlobalOptions(tc.fs)
			o.File = tc.fileFlag
			actual := o.resolvedFile(tc.namespaceFlag)
			if actual != tc.expected {
				t.Fatalf("Expected file: '%s', got: '%s'", tc.expected, actual)
			}
		})
	}
}
