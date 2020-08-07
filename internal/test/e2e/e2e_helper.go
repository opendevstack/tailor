package e2e

import (
	"math/rand"
	"os"
	"os/exec"
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

func setup(t *testing.T) string {
	t.Log("SETUP: Checking for local cluster ...")
	cmd := exec.Command("oc", []string{"whoami"}...)
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Log("SETUP: Local cluster running ...")
	} else if os.Getenv("LAUNCH_LOCAL_CLUSTER") == "yes" {
		launchLocalCluster(t)
	}
	return makeTestProject(t)
}

func teardown(t *testing.T, project string) {
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
