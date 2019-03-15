package openshift

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/utils"
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

func ExportAsTemplate(filter *ResourceFilter, exportOptions *cli.ExportOptions) (string, error) {
	ret := ""
	args := []string{"export", "--as-template=tailor", "--output=yaml"}
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
	outBytes, errBytes, err := cli.RunCmd(cmd)

	if err != nil {
		ret = string(errBytes)
		if strings.Contains(ret, "no resources found") {
			cli.DebugMsg("No", target, "resources found.")
			return "", nil
		}
		return "", fmt.Errorf(
			"Failed to export %s resources.\n"+
				"%s\n",
			target,
			ret,
		)
	}

	cli.DebugMsg("Exported", target, "resources")

	if len(outBytes) == 0 {
		return "", nil
	}

	var f interface{}
	err = yaml.Unmarshal(outBytes, &f)
	if err != nil {
		err = utils.DisplaySyntaxError(outBytes, err)
		return "", err
	}
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

	cli.DebugMsg("Remove metadata from template")
	metadataPointer, _ := gojsonpointer.NewJsonPointer("/metadata")
	_, err = metadataPointer.Delete(m)
	if err != nil {
		cli.DebugMsg("Could not delete metadata from template")
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
	outBytes, errBytes, err := cli.RunCmd(cmd)

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
	containsNamespace, err := templateContainsTailorNamespaceParam(filename)
	if err != nil {
		return []byte{}, err
	}
	if containsNamespace {
		args = append(args, "--param=TAILOR_NAMESPACE="+compareOptions.Namespace)
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
	outBytes, errBytes, err := cli.RunCmd(cmd)

	if len(errBytes) > 0 {
		fmt.Println(string(errBytes))
	}
	if err != nil {
		return []byte{}, err
	}

	cli.DebugMsg("Processed template:", filename)
	return outBytes, err
}

// Returns true if template contains a param like "name: TAILOR_NAMESPACE"
func templateContainsTailorNamespaceParam(filename string) (bool, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return false, nil
	}
	var f interface{}
	err = yaml.Unmarshal(b, &f)
	if err != nil {
		err = utils.DisplaySyntaxError(b, err)
		return false, err
	}
	m := f.(map[string]interface{})
	objectsPointer, _ := gojsonpointer.NewJsonPointer("/parameters")
	items, _, err := objectsPointer.Get(m)
	if err != nil {
		return false, nil
	}
	for _, v := range items.([]interface{}) {
		nameVal := v.(map[string]interface{})["name"]
		paramName := strings.TrimSpace(nameVal.(string))
		if paramName == "TAILOR_NAMESPACE" {
			return true, nil
		}
	}
	return false, nil
}
