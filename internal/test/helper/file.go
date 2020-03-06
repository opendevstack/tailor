package helper

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"
)

// SomeFilesExistFS is a mock filesystem where some files exist.
type SomeFilesExistFS struct {
	Existing []string
}

// Stat always returns a nil error.
func (fs SomeFilesExistFS) Stat(name string) (os.FileInfo, error) {
	for _, ef := range fs.Existing {
		if ef == name {
			return nil, nil
		}
	}
	return nil, os.ErrNotExist
}

// ReadFixtureFile returns the contents of the fixture file or fails.
func ReadFixtureFile(t *testing.T, filename string) []byte {
	return readFileOrFatal(t, "../fixtures/"+filename)
}

// ReadGoldenFile returns the contents of the golden file or fails.
func ReadGoldenFile(t *testing.T, filename string) []byte {
	return readFileOrFatal(t, "../golden/"+filename)
}

func readFileOrFatal(t *testing.T, name string) []byte {
	b, err := readFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readFile(name string) ([]byte, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return []byte{}, fmt.Errorf("Could not get filename when looking for %s", name)
	}
	filepath := path.Join(path.Dir(filename), name)
	return ioutil.ReadFile(filepath)
}
