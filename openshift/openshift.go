package openshift

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/cli"
	"github.com/xeipuuv/gojsonpointer"
)

var (
	KindMapping = map[string]string{
		"svc":                   "Service",
		"service":               "Service",
		"route":                 "Route",
		"dc":                    "DeploymentConfig",
		"deploymentconfig":      "DeploymentConfig",
		"bc":                    "BuildConfig",
		"buildconfig":           "BuildConfig",
		"is":                    "ImageStream",
		"imagestream":           "ImageStream",
		"pvc":                   "PersistentVolumeClaim",
		"persistentvolumeclaim": "PersistentVolumeClaim",
		"template":              "Template",
		"cm":                    "ConfigMap",
		"configmap":             "ConfigMap",
		"secret":                "Secret",
		"rolebinding":           "RoleBinding",
		"serviceaccount":        "ServiceAccount",
	}
)

func ExportAsTemplate(filter *ResourceFilter, name string, exportOptions *cli.ExportOptions) (string, error) {
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
			return "", nil
		}
		fmt.Printf("Failed to export resources: %s.\n", target)
		fmt.Println(fmt.Sprint(err) + ": " + ret)
		return "", err
	}

	cli.DebugMsg("Exported", target, "resources")

	if len(out) == 0 {
		return "", nil
	}

	var f interface{}
	yaml.Unmarshal(out, &f)
	m := f.(map[string]interface{})

	objectsPointer, _ := gojsonpointer.NewJsonPointer("/objects")
	items, _, err := objectsPointer.Get(m)
	if err != nil {
		return "", fmt.Errorf(
			"Could not get objects of exported template: %s", err,
		)
	}
	for k, v := range items.([]interface{}) {
		item, err := NewResourceItem(v.(map[string]interface{}), "platform")
		if err != nil {
			return "", fmt.Errorf(
				"Could not parse object of exported template: %s", err,
			)
		}
		item.RemoveUnmanagedAnnotations()
		itemPointer, _ := gojsonpointer.NewJsonPointer("/objects/" + strconv.Itoa(k))
		itemPointer.Set(m, item.Config)
	}

	b, err := yaml.Marshal(m)
	if err != nil {
		return "", fmt.Errorf(
			"Could not marshal modified template: %s", err,
		)
	}

	return string(b), err
}

func ExportResources(filter *ResourceFilter, compareOptions *cli.CompareOptions) ([]byte, error) {
	target := filter.ConvertToKinds()
	args := []string{"export", target, "--output=yaml"}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		compareOptions.Selector,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outBytes := stdout.Bytes()
	errBytes := stderr.Bytes()

	if err != nil {
		ret := string(errBytes)

		if strings.Contains(ret, "no resources found") {
			cli.DebugMsg("No", target, "resources found.")
			return []byte{}, nil
		}

		return []byte{}, fmt.Errorf(
			"Failed to export %s resources.\n"+
				"%s\n",
			target,
			ret,
		)
	}

	cli.DebugMsg("Exported", target, "resources")
	return outBytes, nil
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

	actualParamFiles := compareOptions.ParamFiles
	if len(actualParamFiles) == 0 {
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
			actualParamFiles = []string{f}
		}
	}
	if len(actualParamFiles) > 0 {
		paramFileBytes := []byte{}
		for _, f := range actualParamFiles {
			cli.DebugMsg("Reading contents of param file", f)
			b, err := ioutil.ReadFile(f)
			if err != nil {
				return []byte{}, err
			}
			paramFileBytes = append(paramFileBytes, b...)
		}
		tempParamFile := ".combined.env"
		defer os.Remove(tempParamFile)
		cli.DebugMsg("Writing contents of param files into", tempParamFile)
		ioutil.WriteFile(tempParamFile, paramFileBytes, 0644)
		paramFileContent := string(paramFileBytes)
		if strings.Contains(paramFileContent, ".ENC=") {
			cli.DebugMsg(tempParamFile, "needs to be decrypted")
			readParams, err := NewParams(paramFileContent, compareOptions.PrivateKey, compareOptions.Passphrase)
			if err != nil {
				return []byte{}, err
			}
			readContent, _ := readParams.Process(true, false)
			tempDecFile := tempParamFile + ".dec"
			defer os.Remove(tempDecFile)
			ioutil.WriteFile(tempDecFile, []byte(readContent), 0644)
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
