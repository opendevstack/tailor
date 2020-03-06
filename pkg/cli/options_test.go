package cli

import (
	"os"
	"testing"
)

type fileNotExistsFS struct {
}

func (fileNotExistsFS) Stat(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }

type fileExistsFS struct {
}

func (fileExistsFS) Stat(name string) (os.FileInfo, error) { return nil, nil }

func TestResolvedFile(t *testing.T) {
	tests := map[string]struct {
		fileFlag      string
		namespaceFlag string
		fs            fileStater
		expected      string
	}{
		"no file flag and no namespace flag given": {
			fileFlag:      "Tailorfile", // default
			namespaceFlag: "",
			fs:            &fileExistsFS{},
			expected:      "Tailorfile",
		},
		"no file flag given but namespace flag given and namespaced file exists": {
			fileFlag:      "Tailorfile", // default
			namespaceFlag: "foo",
			fs:            &fileExistsFS{},
			expected:      "Tailorfile.foo",
		},
		"no file flag given but namespace flag given and namespaced file does not exist": {
			fileFlag:      "Tailorfile", // default
			namespaceFlag: "foo",
			fs:            &fileNotExistsFS{},
			expected:      "Tailorfile",
		},
		"file flag given and no namespace flag given": {
			fileFlag:      "mytailorfile",
			namespaceFlag: "",
			fs:            &fileExistsFS{},
			expected:      "mytailorfile",
		},
		"file flag and namespace flag given": {
			fileFlag:      "mytailorfile",
			namespaceFlag: "foo",
			fs:            &fileExistsFS{},
			expected:      "mytailorfile",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			o := &GlobalOptions{File: tc.fileFlag, fs: tc.fs}
			actual := o.resolvedFile(tc.namespaceFlag)
			if actual != tc.expected {
				t.Fatalf("Expected file: '%s', got: '%s'", tc.expected, actual)
			}
		})
	}
}
