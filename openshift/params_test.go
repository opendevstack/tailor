package openshift

import (
	"io/ioutil"
	"strings"
	"testing"
)

func TestDecryptedParams(t *testing.T) {
	input := readFileContent(t, "test-encrypted.env")
	t.Logf("Read input: %s", input)
	expected := readFileContent(t, "test-cleartext.env")
	t.Logf("Read expected: %s", expected)
	actual, err := DecryptedParams(input, "test-private.key", "")
	if err != nil {
		t.Error(err)
	}
	if actual != expected {
		t.Errorf("Mismatch, got: %v, want: %v.", actual, expected)
	}
}

func TestEncodedParams(t *testing.T) {
	input := readFileContent(t, "test-encrypted.env")
	t.Logf("Read input: %s", input)
	expected := readFileContent(t, "test-encoded.env")
	t.Logf("Read expected: %s", expected)
	actual, err := EncodedParams(input, "test-private.key", "")
	if err != nil {
		t.Error(err)
	}
	if actual != expected {
		t.Errorf("Mismatch, got: %v, want: %v.", actual, expected)
	}
}

func TestEncryptedParams(t *testing.T) {
	previous := readFileContent(t, "test-encrypted.env")
	t.Logf("Read previous: %s", previous)
	input := readFileContent(t, "test-cleartext.env")
	// Add one additional line ...
	input = input + "BAZ=baz\n"
	t.Logf("Read input: %s", input)
	actual, err := EncryptedParams(input, previous, ".", "test-private.key", "")
	if err != nil {
		t.Error(err)
	}
	// The expected output is the first line of the previous file
	// plus one additional line (added above)
	expectedText := strings.TrimSuffix(previous, "\n")
	expectedLines := strings.Split(expectedText, "\n")
	actualText := strings.TrimSuffix(actual, "\n")
	actualLines := strings.Split(actualText, "\n")
	if actualLines[0] != expectedLines[0] {
		t.Errorf("Mismatch, got: %v, want: %v.", actualLines[0], expectedLines[0])
	}
	if strings.HasPrefix("BAZ=", actualLines[0]) {
		t.Errorf("Mismatch, got: %v, want: %v.", actualLines[0], "BAZ=")
	}
}

func readFileContent(t *testing.T, filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Error(err)
	}
	return string(bytes)
}
