package cli

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ClientProcessorExporter allows to process templates and export resources.
type ClientProcessorExporter interface {
	OcClientProcessor
	OcClientExporter
}

// OcClientProcessor is a stop-gap solution only ... should have a better API.
type OcClientProcessor interface {
	Process(args []string) ([]byte, []byte, error)
}

// OcClientExporter allows to export resources.
type OcClientExporter interface {
	Export(target string, label string) ([]byte, error)
}

// OcClientPatcher allows to patch a resource.
type OcClientPatcher interface {
	Patch(target string, patches string) ([]byte, error)
}

// OcClientDeleter allows to delete a resource.
type OcClientDeleter interface {
	Delete(kind string, name string) ([]byte, error)
}

// OcClientCreator allows to create a resource.
type OcClientCreator interface {
	Create(config string, selector string) ([]byte, error)
}

// OpenshiftVersion represents the client/server version pair.
type OpenshiftVersion struct {
	Client string
	Server string
}

// OcClient is a wrapper around the "oc" binary (client).
type OcClient struct {
	namespace string
}

// NewOcClient creates a new ocClient.
func NewOcClient(namespace string) *OcClient {
	return &OcClient{namespace: namespace}
}

// Matches is true when client and server version are equal.
func (ov OpenshiftVersion) Matches() bool {
	return ov.Client == ov.Server
}

// Version returns an OpenshiftVersion.
func (c *OcClient) Version() (OpenshiftVersion, error) {
	cmd := c.execPlainOcCmd([]string{"version"})
	outBytes, errBytes, err := c.runCmd(cmd)
	if err != nil {
		return OpenshiftVersion{"?", "?"}, fmt.Errorf("Failed to query client and server version, got: %s", string(errBytes))
	}
	output := string(outBytes)

	ocClientVersion := ""
	ocServerVersion := ""
	extractVersion := func(versionPart string) string {
		ocVersionParts := strings.SplitN(versionPart, ".", 3)
		return strings.Join(ocVersionParts[:len(ocVersionParts)-1], ".")
	}

	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	for _, line := range lines {
		if len(line) > 0 {
			parts := strings.SplitN(line, " ", 2)
			if parts[0] == "oc" {
				ocClientVersion = extractVersion(parts[1])
			}
			if parts[0] == "openshift" {
				ocServerVersion = extractVersion(parts[1])
			}
		}
	}

	if len(ocClientVersion) == 0 || !strings.Contains(ocClientVersion, ".") {
		ocClientVersion = "?"
	}
	if len(ocServerVersion) == 0 || !strings.Contains(ocServerVersion, ".") {
		ocServerVersion = "?"
	}

	if ocClientVersion == "?" || ocServerVersion == "?" {
		err = fmt.Errorf("Client and server version could not be detected properly, got: %s", output)
	}

	return OpenshiftVersion{Client: ocClientVersion, Server: ocServerVersion}, err
}

// CurrentProject returns the currently active project name (namespace).
func (c *OcClient) CurrentProject() (string, error) {
	cmd := c.execPlainOcCmd([]string{"project", "--short"})
	n, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(n)), err
}

// CheckProjectExists returns true if the given project (namespace) exists.
func (c *OcClient) CheckProjectExists(p string) (bool, error) {
	cmd := c.execPlainOcCmd([]string{"project", p, "--short"})
	_, err := cmd.CombinedOutput()
	return err == nil, err
}

// CheckLoggedIn returns true if the given project (namespace) exists.
func (c *OcClient) CheckLoggedIn() (bool, error) {
	cmd := exec.Command(ocBinary, "whoami")
	_, err := cmd.CombinedOutput()
	return err == nil, err
}

// Process processes an OpenShift template.
// The API is just a stop-gap solution and will be better in the future.
func (c *OcClient) Process(args []string) ([]byte, []byte, error) {
	processArgs := append([]string{"process"}, args...)
	cmd := c.execPlainOcCmd(processArgs)
	return c.runCmd(cmd)
}

// Export exports resources from OpenShift as a template.
func (c *OcClient) Export(target string, label string) ([]byte, error) {
	args := []string{"export", target, "--output=yaml", "--as-template=tailor"}
	cmd := c.execOcCmd(
		args,
		c.namespace,
		label,
	)
	outBytes, errBytes, err := c.runCmd(cmd)

	if err != nil {
		ret := string(errBytes)

		if strings.Contains(ret, "no resources found") {
			return []byte{}, nil
		}

		return []byte{}, fmt.Errorf(
			"Failed to export %s resources.\n"+
				"%s\n",
			target,
			ret,
		)
	}

	return outBytes, nil
}

// Patch applies the given patch on the target.
func (c *OcClient) Patch(target string, patches string) ([]byte, error) {
	args := []string{"patch", target, "--type=json", "--patch", patches}
	cmd := c.execOcCmd(
		args,
		c.namespace,
		"", // empty as name and selector is not allowed
	)
	_, errBytes, err := c.runCmd(cmd)
	return errBytes, err
}

// Create creates given resource configuration.
func (c *OcClient) Create(config string, selector string) ([]byte, error) {
	args := []string{"create", "-f", "-"}
	cmd := c.execOcCmd(
		args,
		c.namespace,
		selector,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	go func() {
		defer stdin.Close()
		_, _ = io.WriteString(stdin, config)
	}()
	_, errBytes, err := c.runCmd(cmd)
	return errBytes, err
}

// Delete deletes given resource.
func (c *OcClient) Delete(kind string, name string) ([]byte, error) {
	args := []string{"delete", kind, name}
	cmd := c.execOcCmd(
		args,
		c.namespace,
		"", // empty as name and selector is not allowed
	)
	_, errBytes, err := c.runCmd(cmd)
	return errBytes, err
}

func (c *OcClient) execOcCmd(args []string, namespace string, selector string) *exec.Cmd {
	if len(namespace) > 0 {
		args = append(args, "--namespace="+namespace)
	}
	if len(selector) > 0 {
		args = append(args, "--selector="+selector)
	}
	return c.execPlainOcCmd(args)
}

func (c *OcClient) execPlainOcCmd(args []string) *exec.Cmd {
	return c.execCmd(ocBinary, args)
}

func (c *OcClient) execCmd(executable string, args []string) *exec.Cmd {
	if verbose {
		PrintBluef("--> %s\n", executable+" "+strings.Join(args, " "))
	}
	return exec.Command(executable, args...)
}

func (c *OcClient) runCmd(cmd *exec.Cmd) (outBytes, errBytes []byte, err error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	outBytes = stdout.Bytes()
	errBytes = stderr.Bytes()
	return outBytes, errBytes, err
}
