package utils

import (
	"io/ioutil"
	"os"
	"path"
)

// ReadFile reads the content of given filename and returns it as a string
func ReadFile(filename string) (string, error) {
	if _, err := os.Stat(filename); err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// AbsoluteOrRelativePath returns p if it is absolute,
// otherwise returns p relative to contextDir.
func AbsoluteOrRelativePath(p string, contextDir string) string {
	if path.IsAbs(p) {
		return p
	}
	if contextDir == "." {
		return p
	}
	return contextDir + "/" + p
}
