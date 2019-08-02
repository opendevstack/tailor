package utils

import (
	"io/ioutil"
	"os"

	"github.com/opendevstack/tailor/pkg/cli"
)

// ReadFile reads the content of given filename and returns it as a string
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
