package openshift

import (
	"bytes"
	"fmt"
	"github.com/opendevstack/tailor/cli"
	"io/ioutil"
	"os"
	"strings"
)

func ExportAsTemplate(filter *ResourceFilter, name string, exportOptions *cli.ExportOptions) ([]byte, error) {
	ret := ""
	args := []string{"export", "--as-template=" + name, "--output=yaml"}
	if len(filter.Label) > 0 {
		args = append(args, "--selector="+filter.Label)
	}
	target := filter.ConvertToTarget()
	args = append(args, target)
	cmd := cli.ExecOcCmd(
		args,
		exportOptions.Namespace,
		exportOptions.Selector,
	)
	out, err := cmd.CombinedOutput()

	if err != nil {
		ret = string(out)
		if strings.Contains(ret, "no resources found") {
			cli.DebugMsg("No resources '" + target + "' found.")
			return []byte{}, nil
		}
		fmt.Printf("Failed to export resources: %s.\n", target)
		fmt.Println(fmt.Sprint(err) + ": " + ret)
		return nil, err
	}

	cli.DebugMsg("Exported", target, "resources")
	return out, err
}

func ExportResources(filter *ResourceFilter, compareOptions *cli.CompareOptions) ([]byte, error) {
	ret := ""
	target := filter.ConvertToKinds()
	args := []string{"export", target, "--output=yaml"}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		compareOptions.Selector,
	)
	out, err := cmd.CombinedOutput()

	if err != nil {
		ret = string(out)
		if strings.Contains(ret, "no resources found") {
			cli.DebugMsg("No", target, "resources found.")
			return []byte{}, nil
		}
		fmt.Printf("Failed to export %s resources.\n", target)
		fmt.Println(fmt.Sprint(err) + ": " + ret)
		return nil, err
	}

	cli.DebugMsg("Exported", target, "resources")
	return out, err
}

func ProcessTemplate(templateDir string, name string, paramDir string, compareOptions *cli.CompareOptions) ([]byte, error) {
	filename := templateDir + string(os.PathSeparator) + name

	args := []string{"process", "--filename=" + filename, "--output=yaml"}

	if len(compareOptions.Labels) > 0 {
		args = append(args, "--labels="+compareOptions.Labels)
	}

	for _, param := range compareOptions.Params {
		args = append(args, "--param="+param)
	}

	actualParamFile := compareOptions.ParamFile
	if len(actualParamFile) == 0 {
		// Prefer <namespace> folder over current directory
		if paramDir == "." {
			if _, err := os.Stat(compareOptions.Namespace); err == nil {
				paramDir = compareOptions.Namespace
			}
		}

		cli.DebugMsg(fmt.Sprintf("Looking for param files in '%s'", paramDir))

		fileParts := strings.Split(name, ".")
		fileParts[len(fileParts)-1] = "env"
		f := paramDir + string(os.PathSeparator) + strings.Join(fileParts, ".")
		if _, err := os.Stat(f); err == nil {
			actualParamFile = f
		}
	}
	if len(actualParamFile) > 0 {
		tempParamFile := actualParamFile
		b, err := ioutil.ReadFile(actualParamFile)
		if err != nil {
			return []byte{}, err
		}
		paramFileContent := string(b)
		if strings.Contains(paramFileContent, ".ENC=") {
			cli.DebugMsg(actualParamFile, "needs to be decrypted")
			readParams, err := NewParams(paramFileContent, compareOptions.PrivateKey, compareOptions.Passphrase)
			if err != nil {
				return []byte{}, err
			}
			readContent, _ := readParams.Process(true, false)
			tempParamFile = actualParamFile + ".dec"
			defer os.Remove(tempParamFile)
			ioutil.WriteFile(tempParamFile, []byte(readContent), 0644)
		}
		args = append(args, "--param-file="+tempParamFile)
	}

	if compareOptions.IgnoreUnknownParameters {
		args = append(args, "--ignore-unknown-parameters=true")
	}
	cmd := cli.ExecPlainOcCmd(args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outBytes := stdout.Bytes()
	errBytes := stderr.Bytes()

	if len(errBytes) > 0 {
		fmt.Println(string(errBytes))
	}
	if err != nil {
		return []byte{}, err
	}

	cli.DebugMsg("Processed template:", filename)
	return outBytes, err
}

func UpdateRemote(changeset *Changeset, compareOptions *cli.CompareOptions) error {
	for _, change := range changeset.Create {
		ocApply(change, "Creating", compareOptions)
	}

	for _, change := range changeset.Delete {
		ocDelete(change, compareOptions)
	}

	for _, change := range changeset.Update {
		ocApply(change, "Updating", compareOptions)
	}

	return nil
}

func ocDelete(change *Change, compareOptions *cli.CompareOptions) error {
	kind := change.Kind
	name := change.Name
	fmt.Println("Deleting", kind, name)
	args := []string{"delete", kind, name}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		"", // empty as name and selector is not allowed
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("Removed '%s/%s'.\n", kind, name)
	} else {
		fmt.Printf("Failed to remove '%s/%s' - aborting.\n", kind, name)
		fmt.Println(fmt.Sprint(err) + ": " + string(out))
		return err
	}
	return nil
}

func ocApply(change *Change, action string, compareOptions *cli.CompareOptions) error {
	kind := change.Kind
	name := change.Name
	config := change.DesiredState
	fmt.Println(action, kind, name)
	ioutil.WriteFile(".PROCESSED_TEMPLATE", []byte(config), 0644)

	args := []string{"apply", "--filename=" + ".PROCESSED_TEMPLATE"}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		compareOptions.Selector,
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("Applied processed '%s' template.\n", kind)
		os.Remove(".PROCESSED_TEMPLATE")
	} else {
		fmt.Printf("Failed to apply processed '%s' template - aborting.\n", kind)
		fmt.Println("It is left for inspection at .PROCESSED_TEMPLATE.")
		fmt.Println(fmt.Sprint(err) + ": " + string(out))
		return err
	}
	return nil
}
