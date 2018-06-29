package openshift

import (
	"bytes"
	"fmt"
	"github.com/michaelsauter/ocdiff/cli"
	"io/ioutil"
	"os"
	"strings"
)

func ExportAsTemplate(filter *ResourceFilter, name string) ([]byte, error) {
	ret := ""
	args := []string{"export", "--as-template=" + name, "--output=yaml"}
	if len(filter.Label) > 0 {
		args = append(args, "--selector="+filter.Label)
	}
	target := filter.ConvertToTarget()
	args = append(args, target)
	cmd := cli.ExecOcCmd(args)
	out, err := cmd.CombinedOutput()

	if err != nil {
		ret = string(out)
		if strings.Contains(ret, "no resources found") {
			cli.VerboseMsg("No resources '" + target + "' found.")
			return []byte{}, nil
		}
		fmt.Printf("Failed to export resources: %s.\n", target)
		fmt.Println(fmt.Sprint(err) + ": " + ret)
		return nil, err
	}

	cli.VerboseMsg("Exported", target, "resources")
	return out, err
}

func ExportResources(filter *ResourceFilter) ([]byte, error) {
	ret := ""
	target := filter.ConvertToTarget()
	args := []string{"export", target, "--output=yaml"}
	cmd := cli.ExecOcCmd(args)
	out, err := cmd.CombinedOutput()

	if err != nil {
		ret = string(out)
		if strings.Contains(ret, "no resources found") {
			cli.VerboseMsg("No", target, "resources found.")
			return []byte{}, nil
		}
		fmt.Printf("Failed to export %s resources.\n", target)
		fmt.Println(fmt.Sprint(err) + ": " + ret)
		return nil, err
	}

	cli.VerboseMsg("Exported", target, "resources")
	return out, err
}

func ProcessTemplate(templateDir string, name string, paramDir string, label string, params []string, paramFile string, ignoreUnknownParameters bool, privateKey string, passphrase string) ([]byte, error) {
	filename := templateDir + string(os.PathSeparator) + name
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		cli.VerboseMsg("Template '" + filename + "' does not exist.")
		return []byte{}, nil
	}

	args := []string{"process", "--filename=" + filename, "--output=yaml"}

	if len(label) > 0 {
		args = append(args, "--labels="+label)
	}

	for _, param := range params {
		args = append(args, "--param="+param)
	}

	actualParamFile := paramFile
	if len(actualParamFile) == 0 {
		// Prefer <namespace> folder over current directory
		if paramDir == "." {
			if namespaceDir, err := cli.GetOcNamespace(); err == nil {
				if _, err := os.Stat(namespaceDir); err == nil {
					paramDir = namespaceDir
				}
			}
		}

		cli.VerboseMsg(fmt.Sprintf("Looking for param files in '%s'", paramDir))

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
			cli.VerboseMsg(actualParamFile, "needs to be decrypted")
			readParams, err := NewParams(paramFileContent, privateKey, passphrase)
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

	if ignoreUnknownParameters {
		args = append(args, "--ignore-unknown-parameters=true")
	}
	cmd := cli.ExecPlainOcCmd(args)
	out, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Failed to process template: %s.\n", filename)
		fmt.Println(fmt.Sprint(err) + ": " + string(out))
		return []byte{}, err
	}

	cli.VerboseMsg("Processed template:", filename)
	return out, err
}

func UpdateRemote(changeset *Changeset) error {
	for _, change := range changeset.Create {
		ocApply(change, "Creating")
	}

	for _, change := range changeset.Delete {
		ocDelete(change)
	}

	for _, change := range changeset.Update {
		ocApply(change, "Updating")
	}

	return nil
}

func ocDelete(change *Change) error {
	kind := change.Kind
	name := change.Name
	fmt.Println("Deleting", kind, name)
	args := []string{"delete", kind, name}
	cmd := cli.ExecOcCmd(args)
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

func ocApply(change *Change, action string) error {
	kind := change.Kind
	name := change.Name
	config := change.DesiredState
	fmt.Println(action, kind, name)
	b := []byte(config)
	unescapedRaw := bytes.Replace(b, []byte("%%"), []byte("%"), -1)
	ioutil.WriteFile(".PROCESSED_TEMPLATE", unescapedRaw, 0644)

	args := []string{"apply", "--filename=" + ".PROCESSED_TEMPLATE"}
	cmd := cli.ExecOcCmd(args)
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
