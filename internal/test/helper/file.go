package helper

import (
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"testing"
)

// ReadFixtureFileOrErr returns the contents of the fixture file or an error.
func ReadFixtureFileOrErr(filename string) ([]byte, error) {
	return readFileOrErr("../fixtures/" + filename)
}

// ReadGoldenFileOrErr returns the contents of the golden file or an error.
func ReadGoldenFileOrErr(t *testing.T, filename string) ([]byte, error) {
	return readFileOrErr("../golden/" + filename)
}

// ReadFixtureFile returns the contents of the fixture file or fails.
func ReadFixtureFile(t *testing.T, filename string) []byte {
	return readFile(t, "../fixtures/"+filename)
}

// ReadGoldenFile returns the contents of the golden file or fails.
func ReadGoldenFile(t *testing.T, filename string) []byte {
	return readFile(t, "../golden/"+filename)
}

func readFile(t *testing.T, name string) []byte {
	b, err := readFileOrErr(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readFileOrErr(name string) ([]byte, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return []byte{}, fmt.Errorf("Could not get filename when looking for %s", name)
	}
	filepath := path.Join(path.Dir(filename), name)
	return ioutil.ReadFile(filepath)
}
