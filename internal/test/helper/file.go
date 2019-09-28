package helper

import (
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"testing"
)

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
