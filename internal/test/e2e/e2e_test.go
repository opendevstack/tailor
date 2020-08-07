package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
)

type testCaseSteps []testCaseStep

type testCaseStep struct {
	Command       string                       `json:"command"`
	WantStdout    bool                         `json:"wantStdout"`
	WantStderr    bool                         `json:"wantStderr"`
	WantErr       bool                         `json:"wantErr"`
	WantResources map[string]bool              `json:"wantResources"`
	WantFields    map[string]map[string]string `json:"wantFields"`
}

type outputData struct {
	Project string
}

func TestE2E(t *testing.T) {
	testProjectName := setup(t, false)
	defer teardown(t, testProjectName, false)

	err := os.Chdir("testdata")
	if err != nil {
		t.Fatalf("Fail to chdir to testdata: %s", err)
	}

	tailorBinary := getTailorBinary()

	walkSubdirs(t, ".", func(subdir string) {
		runTestCase(t, testProjectName, tailorBinary, subdir)
	})
}

func walkSubdirs(t *testing.T, dir string, fun func(subdir string)) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.IsDir() {
			fun(f.Name())
		}
	}
}

func runTestCase(t *testing.T, testProjectName string, tailorBinary string, testCase string) {
	t.Log("Running steps for test case", testCase)
	tcs, err := readTestCaseSteps(testCase)
	if err != nil {
		t.Fatal(err)
	}

	for i, tc := range tcs {
		t.Run(fmt.Sprintf("%s step #%d", testCase, i), func(t *testing.T) {
			stepDir := fmt.Sprintf("%s/%d", testCase, i)
			templateData := outputData{
				Project: testProjectName,
			}
			args := []string{
				"--non-interactive",
				"-n", testProjectName,
				"--template-dir", stepDir,
			}
			args = append(args, strings.Split(tc.Command, " ")...)
			t.Logf("Running tailor with: %s", strings.Join(args, " "))
			gotStdout, gotStderr, gotErr := runCmd(tailorBinary, args)
			checkErr(t, tc.WantErr, gotErr, gotStderr)
			checkStderr(t, tc.WantStderr, gotStderr, templateData, stepDir)
			checkStdout(t, tc.WantStdout, gotStdout, templateData, stepDir)
			checkResources(t, tc.WantResources, testProjectName)
			checkFields(t, tc.WantFields, testProjectName)
		})
	}

}

func checkErr(t *testing.T, wantErr bool, gotErr error, gotStderr []byte) {
	if wantErr {
		if gotErr == nil {
			t.Fatal("Want error, got none")
		}
	} else {
		if gotErr != nil {
			t.Fatalf("Got error: %s: %s", gotErr, gotStderr)
		}
	}
}

func checkStderr(t *testing.T, wantStderr bool, gotStderr []byte, templateData outputData, stepDir string) {
	if wantStderr {
		var wantStderr bytes.Buffer
		tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/want.err", stepDir)))
		err := tmpl.Execute(&wantStderr, templateData)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(wantStderr.Bytes(), gotStderr); diff != "" {
			t.Fatalf("Stderr mismatch (-want +got):\n%s", diff)
		}
	}
}

func checkStdout(t *testing.T, wantStdout bool, gotStdout []byte, templateData outputData, stepDir string) {
	if wantStdout {
		var wantStdout bytes.Buffer
		tmpl := template.Must(template.ParseFiles(fmt.Sprintf("%s/want.out", stepDir)))
		err := tmpl.Execute(&wantStdout, templateData)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(wantStdout.Bytes(), gotStdout); diff != "" {
			t.Fatalf("Stdout mismatch (-want +got):\n%s", diff)
		}
	}
}

func checkResources(t *testing.T, wantResources map[string]bool, projectName string) {
	for res, wantExists := range wantResources {
		_, _, err := runCmd("oc", []string{"-n", projectName, "get", res})
		gotExists := err == nil
		if gotExists != wantExists {
			t.Fatalf("Resource %s: want exists=%t, got exists=%t\n", res, wantExists, gotExists)
		}
	}
}

func checkFields(t *testing.T, wantFields map[string]map[string]string, projectName string) {
	for res, jsonPaths := range wantFields {
		for jsonPath, wantVal := range jsonPaths {
			gotVal, _, err := runCmd("oc", []string{
				"-n", projectName,
				"get", res,
				"-o", fmt.Sprintf("jsonpath={%s}", jsonPath),
			})
			if err != nil {
				t.Fatalf("Could not get path %s of resource %s: %s", jsonPath, res, err)
			}
			if string(gotVal) != wantVal {
				t.Fatalf("Field %s %s: want val=%s, got val=%s\n", res, jsonPath, wantVal, gotVal)
			}
		}
	}
}

func runCmd(executable string, args []string) (outBytes, errBytes []byte, err error) {
	cmd := exec.Command(executable, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	outBytes = stdout.Bytes()
	errBytes = stderr.Bytes()
	return outBytes, errBytes, err
}

func readTestCaseSteps(folder string) (testCaseSteps, error) {
	content, err := ioutil.ReadFile(folder + "/steps.json")
	if err != nil {
		return nil, fmt.Errorf("Cannot read file: %w", err)
	}

	var tc testCaseSteps
	err = json.Unmarshal(content, &tc)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse JSON: %w", err)
	}
	return tc, nil
}
