package utils

import (
	"io/ioutil"
	"os"
	"path"
)

// FileStater is a helper interface to allow testing.
type FileStater interface {
	Stat(name string) (os.FileInfo, error)
}

// OsFS implements Stat() for local disk.
type OsFS struct{}

// Stat proxies to os.Stat.
func (OsFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

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
