package openshift

import (
	"bytes"
	"fmt"
	"github.com/michaelsauter/ocdiff/cli"
	"io/ioutil"
	"os"
	"strings"
)

func ExportAsTemplate(filter *ResourceFilter) ([]byte, error) {
	ret := ""
	args := []string{"export", "--as-template=" + filter.Kind, "--output=yaml"}
	if len(filter.Label) > 0 {
		args = append(args, "--selector="+filter.Label)
	}
	if len(filter.Names) > 0 {
		for _, name := range filter.Names {
			args = append(args, filter.Kind+"/"+name)
		}
	} else {
		args = append(args, filter.Kind)
	}
	cmd := cli.ExecOcCmd(args)
	out, err := cmd.CombinedOutput()

	if err != nil {
		ret = string(out)
		if strings.Contains(ret, "no resources found") {
			cli.VerboseMsg("No resource '" + filter.Kind + "' found.")
			return []byte{}, nil
		}
		fmt.Printf("Failed to export resource: %s.\n", filter.Kind)
		fmt.Println(fmt.Sprint(err) + ": " + ret)
		return nil, err
	}

	cli.VerboseMsg("Exported", filter.Kind, "resources")
	return out, err
}

func ExportResource(kind string) ([]byte, error) {
	ret := ""
	args := []string{"export", kind, "--output=yaml"}
	cmd := cli.ExecOcCmd(args)
	out, err := cmd.CombinedOutput()

	if err != nil {
		ret = string(out)
		if strings.Contains(ret, "no resources found") {
			cli.VerboseMsg("No", kind, "resources found.")
			return []byte{}, nil
		}
		fmt.Printf("Failed to export %s resources.\n", kind)
		fmt.Println(fmt.Sprint(err) + ": " + ret)
		return nil, err
	}

	cli.VerboseMsg("Exported", kind, "resources")
	return out, err
}

func ProcessTemplate(templateDir string, name string, paramDir string, label string, params []string, paramFile string, ignoreUnknownParameters bool, privateKey string) ([]byte, error) {
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
			// TODO: Use already read file contents to avoid reading twice.
			decrypted, err := cli.ReadEnvFile(actualParamFile, privateKey)
			if err != nil {
				return []byte{}, err
			}
			decrypted = strings.Replace(decrypted, ".ENC=", "=", -1)
			tempParamFile = actualParamFile + ".dec"
			defer os.Remove(tempParamFile)
			ioutil.WriteFile(tempParamFile, []byte(decrypted), 0644)
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

func UpdateRemote(changesets map[string]*Changeset) error {
	for kind, changeset := range changesets {
		for name, configs := range changeset.Create {
			fmt.Println("Creating", kind, name)
			ocApply(kind, name, configs[1])
		}

		for name, _ := range changeset.Delete {
			fmt.Println("Deleting", kind, name)
			ocDelete(kind, name)
		}

		for name, configs := range changeset.Update {
			fmt.Println("Updating", kind, name)
			ocApply(kind, name, configs[1])
		}
	}

	return nil
}

func ocDelete(kind string, name string) error {
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

func ocApply(kind string, name string, config string) error {
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
