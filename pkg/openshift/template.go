package openshift

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/utils"
	"github.com/xeipuuv/gojsonpointer"
)

// ExportAsTemplateFile exports resources in template format.
func ExportAsTemplateFile(filter *ResourceFilter, withAnnotations bool, ocClient cli.OcClientExporter) (string, error) {
	outBytes, err := ocClient.Export(filter.ConvertToKinds(), filter.Label)
	if err != nil {
		return "", err
	}
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

		if !withAnnotations {
			cli.DebugMsg("Remove annotations from item")
			annotationsPointer, _ := gojsonpointer.NewJsonPointer("/metadata/annotations")
			_, err = annotationsPointer.Delete(item.Config)
			if err != nil {
				cli.DebugMsg("Could not delete annotations from item")
			}
		}

		itemPointer, _ := gojsonpointer.NewJsonPointer("/objects/" + strconv.Itoa(k))
		_, _ = itemPointer.Set(m, item.Config)
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

// ProcessTemplate processes template "name" in "templateDir".
func ProcessTemplate(templateDir string, name string, paramDir string, compareOptions *cli.CompareOptions, ocClient cli.OcClientProcessor) ([]byte, error) {
	filename := templateDir + string(os.PathSeparator) + name

	args := []string{"--filename=" + filename, "--output=yaml"}

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

	actualParamFiles := calculateParamFiles(name, paramDir, compareOptions)

	// Now turn the param files into arguments for the oc binary
	if len(actualParamFiles) > 0 {
		paramFileBytes, err := readParamFileBytes(
			actualParamFiles,
			compareOptions.PrivateKey,
			compareOptions.Passphrase,
		)
		if err != nil {
			return []byte{}, err
		}
		tempParamFile := ".combined.env"
		defer os.Remove(tempParamFile)
		cli.DebugMsg("Writing contents of param files into", tempParamFile)
		err = ioutil.WriteFile(tempParamFile, paramFileBytes, 0644)
		if err != nil {
			return []byte{}, err
		}
		args = append(args, "--param-file="+tempParamFile)
	}

	if compareOptions.IgnoreUnknownParameters {
		args = append(args, "--ignore-unknown-parameters=true")
	}
	outBytes, errBytes, err := ocClient.Process(args)

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

func calculateParamFiles(name string, paramDir string, compareOptions *cli.CompareOptions) []string {
	files := compareOptions.ParamFiles
	// If param-file is not given, we assume a param-dir
	if len(files) == 0 {
		// Prefer <namespace> folder over current directory
		if paramDir == "." {
			if _, err := os.Stat(compareOptions.Namespace); err == nil {
				paramDir = compareOptions.Namespace
			}
		}

		cli.DebugMsg(fmt.Sprintf("Looking for param files in '%s'", paramDir))

		fileParts := strings.Split(name, ".")
		fileParts[len(fileParts)-1] = "env"
		f := strings.Join(fileParts, ".")
		if paramDir != "." {
			f = paramDir + string(os.PathSeparator) + f
		}
		if compareOptions.FileExists(f) {
			files = []string{f}
		}
	}
	// Add <namespace>.env file if it exists
	namespaceDotEnvFile := fmt.Sprintf("%s.env", compareOptions.Namespace)
	if !utils.Includes(files, namespaceDotEnvFile) {
		if compareOptions.FileExists(namespaceDotEnvFile) {
			cli.DebugMsg(fmt.Sprintf("Adding param file '%s' by convention", namespaceDotEnvFile))
			files = append(files, namespaceDotEnvFile)
		}
	}
	return files
}

func readParamFileBytes(paramFiles []string, privateKey string, passphrase string) ([]byte, error) {
	paramFileBytes := []byte{}
	for _, f := range paramFiles {
		cli.DebugMsg("Reading content of param file", f)
		b, err := ioutil.ReadFile(f)
		if err != nil {
			return []byte{}, err
		}
		eol := []byte("\n")
		if !bytes.HasSuffix(b, eol) {
			b = append(b, eol...)
		}
		paramFileBytes = append(paramFileBytes, b...)
		// Check if encrypted param file exists, and if so, decrypt and
		// append its content
		encFile := f + ".enc"
		if _, err := os.Stat(encFile); err == nil {
			cli.DebugMsg("Reading content of encrypted param file", encFile)
			b, err := ioutil.ReadFile(encFile)
			if err != nil {
				return []byte{}, err
			}
			encoded, err := EncodedParams(string(b), privateKey, passphrase)
			if err != nil {
				return []byte{}, err
			}
			paramFileBytes = append(paramFileBytes, []byte(encoded)...)
		}
	}
	return paramFileBytes, nil
}
