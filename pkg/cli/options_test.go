package cli

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestNewCompareOptionsExcludes(t *testing.T) {
	tests := map[string]struct {
		excludeFlag  []string
		wantExcludes []string
	}{
		"none": {
			excludeFlag:  []string{},
			wantExcludes: []string{},
		},
		"passed once": {
			excludeFlag:  []string{"bc"},
			wantExcludes: []string{"bc"},
		},
		"passed once comma-separated": {
			excludeFlag:  []string{"bc,is"},
			wantExcludes: []string{"bc", "is"},
		},
		"passed multiple times": {
			excludeFlag:  []string{"bc", "is"},
			wantExcludes: []string{"bc", "is"},
		},
		"passed multiple times and comma-separated": {
			excludeFlag:  []string{"bc,is", "route"},
			wantExcludes: []string{"bc", "is", "route"},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			o, err := NewGlobalOptions(false, "Tailorfile", false, false, false, "oc", false)
			if err != nil {
				t.Fatal(err)
			}
			got, err := NewCompareOptions(
				o,
				"",
				"",
				tc.excludeFlag,
				".",
				".",
				"",
				"",
				"",
				"",
				[]string{},
				[]string{},
				[]string{},
				false,
				false,
				false,
				false,
				false,
				false,
				"")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.wantExcludes, got.Excludes); diff != "" {
				t.Errorf("Compare options mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewExportOptionsExcludes(t *testing.T) {
	tests := map[string]struct {
		excludeFlag  []string
		wantExcludes []string
	}{
		"none": {
			excludeFlag:  []string{},
			wantExcludes: []string{},
		},
		"passed once": {
			excludeFlag:  []string{"bc"},
			wantExcludes: []string{"bc"},
		},
		"passed once comma-separated": {
			excludeFlag:  []string{"bc,is"},
			wantExcludes: []string{"bc", "is"},
		},
		"passed multiple times": {
			excludeFlag:  []string{"bc", "is"},
			wantExcludes: []string{"bc", "is"},
		},
		"passed multiple times and comma-separated": {
			excludeFlag:  []string{"bc,is", "route"},
			wantExcludes: []string{"bc", "is", "route"},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			o, err := NewGlobalOptions(false, "Tailorfile", false, false, false, "oc", false)
			if err != nil {
				t.Fatal(err)
			}
			got, err := NewExportOptions(
				o,
				"",
				"",
				tc.excludeFlag,
				".",
				".",
				false,
				false,
				[]string{},
				"")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.wantExcludes, got.Excludes); diff != "" {
				t.Errorf("Export options mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
