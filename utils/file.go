package utils

import (
	"io/ioutil"
	"os"

	"github.com/opendevstack/tailor/cli"
)

// IncludesPrefix checks if needle is in haystack
func ReadFile(filename string) (string, error) {
	cli.DebugMsg("Reading file", filename)
	if _, err := os.Stat(filename); err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
