package e2e

import (
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func setup(t *testing.T, withTemplatesFolder bool) string {
	t.Log("SETUP: Checking for local cluster ...")
	cmd := exec.Command("oc", []string{"whoami"}...)
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Log("SETUP: Local cluster running ...")
	} else if os.Getenv("LAUNCH_LOCAL_CLUSTER") == "yes" {
		launchLocalCluster(t)
	}
	if withTemplatesFolder {
		makeTemplateFolder(t)
	}
	return makeTestProject(t)
}

func teardown(t *testing.T, project string, withTemplatesFolder bool) {
	if withTemplatesFolder {
		cleanupTemplateFolder(t)
	}
	deleteTestProject(t, project)
}

func getTailorBinary() string {
	dir, _ := os.Getwd()
	return dir + "/../tailor-test"
}

func launchLocalCluster(t *testing.T) {
	t.Log("SETUP: Launching local cluster ...")
	cmd := exec.Command("oc", []string{"cluster", "up"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SETUP: Could not launch local cluster: %s", out)
	}
	t.Log("SETUP: Local cluster launched")
}

func deleteTestProject(t *testing.T, project string) {
	cmd := exec.Command("oc", []string{"delete", "project", project}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("TEARDOWN: Could not delete project %s: %s", project, out)
	}
	t.Log("TEARDOWN:", project, "project deleted")
}

func makeTestProject(t *testing.T) string {
	project := "tailor-e2e-test-" + randomString(6)
	cmd := exec.Command("oc", []string{"new-project", project}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SETUP: Could not create project %s: %s", project, out)
	}
	t.Log("SETUP: Project", project, "created")
	return project
}

func makeTemplateFolder(t *testing.T) {
	err := os.MkdirAll("templates", os.ModePerm)
	if err != nil {
		t.Fatalf("SETUP: Fail to mkdir templates: %s", err)
	}
	err = os.Chdir("templates")
	if err != nil {
		t.Fatalf("SETUP: Fail to chdir templates: %s", err)
	}
	t.Log("SETUP: templates folder created")
}

func cleanupTemplateFolder(t *testing.T) {
	dir, _ := os.Getwd()
	err := os.Chdir(strings.TrimSuffix(dir, "/templates"))
	if err != nil {
		t.Fatalf("TEARDOWN: Fail to chdir templates: %s", err)
	}
	err = os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("TEARDOWN: Could not remove templates folder: %s", err)
	}
	t.Log("TEARDOWN: templates folder removed")
}

func runApply(t *testing.T, tailorBinary string, testProjectName string, tailorParams []string) {
	t.Log("Updating project", testProjectName)
	args := []string{"-n", testProjectName, "apply", "--non-interactive"}
	args = append(args, tailorParams...)
	cmd := exec.Command(tailorBinary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not update project %s: %s", testProjectName, out)
	}
	t.Log("Updated project", testProjectName)
}

func diffWithNoExpectedDrift(t *testing.T, tailorBinary string, testProjectName string, tailorParams []string) {
	t.Log("Calculating diff ...")
	args := []string{"-n", testProjectName, "diff"}
	args = append(args, tailorParams...)
	cmd := exec.Command(tailorBinary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not get status in project %s: %s", testProjectName, out)
	}
	t.Log("Got status in", testProjectName, "project (should have no drift)")
	if !strings.Contains(string(out), "0 to create") {
		t.Fatalf("No resource should be to create. Got: %s", out)
	}
	if !strings.Contains(string(out), "0 to update") {
		t.Fatalf("No resource should be to update. Got: %s", out)
	}
	if !strings.Contains(string(out), "0 to delete") {
		t.Fatalf("No resource should be to delete. Got: %s", out)
	}
	if !strings.Contains(string(out), "in sync") {
		t.Fatalf("Some resources should be in sync. Got: %s", out)
	}
}

func runExport(t *testing.T, tailorBinary string, testProjectName string) {
	cmd := exec.Command(tailorBinary, []string{"-n", testProjectName, "export"}...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Could not export resources in project %s: %s", testProjectName, out)
	}
	err = ioutil.WriteFile("test-template.yml", out, 0644)
	if err != nil {
		t.Fatalf("Fail to write file cm-template.yml: %s", err)
	}
	t.Log("Resources in", testProjectName, "project exported")
}
