package main

import (
	"errors"
	"fmt"
	"github.com/alecthomas/kingpin"
	"github.com/ghodss/yaml"
	"github.com/michaelsauter/ocdiff/cli"
	"github.com/michaelsauter/ocdiff/openshift"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

var (
	app         = kingpin.New("ocdiff", "OC Diff Tool").DefaultEnvars()
	verboseFlag = app.Flag(
		"verbose",
		"Enable verbose output.",
	).Short('v').Bool()
	nonInteractiveFlag = app.Flag(
		"non-interactive",
		"Disable interactive mode.",
	).Bool()

	namespaceFlag = app.Flag(
		"namespace",
		"Namespace (omit to use current)",
	).Short('n').String()
	selectorFlag = app.Flag(
		"selector",
		"Selector (label query) to filter on",
	).Short('l').String()
	templateDirFlag = app.Flag(
		"template-dir",
		"Path to local templates",
	).Short('t').Default(".").Strings()
	paramDirFlag = app.Flag(
		"param-dir",
		"Path to param files for local templates",
	).Short('p').Default(".").Strings()
	publicKeyDirFlag = app.Flag(
		"public-key-dir",
		"Path to public key files",
	).Default(".").String()
	privateKeyFlag = app.Flag(
		"private-key",
		"Path to private key file",
	).Default("private.key").String()

	versionCommand = app.Command(
		"version",
		"Show version",
	)

	statusCommand = app.Command(
		"status",
		"Show diff between remote and local",
	)
	statusLabelsFlag = statusCommand.Flag(
		"labels",
		"Label to set in all resources for this template.",
	).String()
	statusParamFlag = statusCommand.Flag(
		"param",
		"Specify a key-value pair (eg. -p FOO=BAR) to set/override a parameter value in the template.",
	).Strings()
	statusParamFileFlag = statusCommand.Flag(
		"param-file",
		"File containing template parameter values to set/override in the template.",
	).String()
	statusIgnoreUnknownParametersFlag = statusCommand.Flag(
		"ignore-unknown-parameters",
		"If true, will not stop processing if a provided parameter does not exist in the template.",
	).Bool()
	statusUpsertOnlyFlag = statusCommand.Flag(
		"upsert-only",
		"Don't delete resource, only create / update.",
	).Short('u').Bool()
	statusResourceArg = statusCommand.Arg(
		"resource", "Remote resource (defaults to all)",
	).String()

	updateCommand = app.Command(
		"update",
		"Update remote with local",
	)
	updateLabelsFlag = updateCommand.Flag(
		"labels",
		"Label to set in all resources for this template.",
	).String()
	updateParamFlag = updateCommand.Flag(
		"param",
		"Specify a key-value pair (eg. -p FOO=BAR) to set/override a parameter value in the template.",
	).Strings()
	updateParamFileFlag = updateCommand.Flag(
		"param-file",
		"File containing template parameter values to set/override in the template.",
	).String()
	updateIgnoreUnknownParametersFlag = updateCommand.Flag(
		"ignore-unknown-parameters",
		"If true, will not stop processing if a provided parameter does not exist in the template.",
	).Bool()
	updateUpsertOnlyFlag = updateCommand.Flag(
		"upsert-only",
		"Don't delete resource, only create / update.",
	).Short('u').Bool()
	updateResourceArg = updateCommand.Arg(
		"resource", "Remote resource (defaults to all)",
	).String()

	exportCommand = app.Command(
		"export",
		"Export remote state as template",
	)
	exportWriteFilesByKindFlag = exportCommand.Flag(
		"write-files-by-kind",
		"Write export into one template file per kind.",
	).Short('w').Bool()
	exportResourceArg = exportCommand.Arg(
		"resource", "Remote resource (defaults to all)",
	).String()

	secretsCommand = app.Command(
		"secrets",
		"Work with secrets",
	)
	editCommand = secretsCommand.Command(
		"edit",
		"Edit param file",
	)
	editFileArg = editCommand.Arg(
		"file", "File to edit",
	).String()

	reEncryptCommand = secretsCommand.Command(
		"re-encrypt",
		"Re-Encrypt param file(s)",
	)

	revealCommand = secretsCommand.Command(
		"reveal",
		"Show param file contens with revealed secrets",
	)
	revealFileArg = revealCommand.Arg(
		"file", "File to show",
	).String()

	kindMapping = map[string]string{
		"svc":              "Service",
		"service":          "Service",
		"route":            "Route",
		"dc":               "DeploymentConfig",
		"deploymentconfig": "DeploymentConfig",
		"bc":               "BuildConfig",
		"buildconfig":      "BuildConfig",
		"is":               "ImageStream",
		"imagestream":      "ImageStream",
		"pvc":              "PersistentVolumeClaim",
		"persistentvolumeclaim": "PersistentVolumeClaim",
		"template":              "Template",
		"cm":                    "ConfigMap",
		"configmap":             "ConfigMap",
		"secret":                "Secret",
		"rolebinding":           "RoleBinding",
		"serviceaccount":        "ServiceAccount",
	}

	kindToShortMapping = map[string]string{
		"Service":               "svc",
		"Route":                 "route",
		"DeploymentConfig":      "dc",
		"BuildConfig":           "bc",
		"ImageStream":           "is",
		"PersistentVolumeClaim": "pvc",
		"Template":              "template",
		"ConfigMap":             "cm",
		"Secret":                "secret",
		"RoleBinding":           "rolebinding",
		"ServiceAccount":        "serviceaccount",
	}
)

func main() {
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	cli.SetOptions(*verboseFlag, *namespaceFlag, *selectorFlag)

	paramDir := *paramDirFlag
	if (len(paramDir) > 1 || paramDir[0] != ".") && (len(*statusParamFileFlag) > 0 || len(*updateParamFileFlag) > 0) {
		log.Fatalln("You cannot specify both --param-dir and --param-flag.")
	}

	switch command {
	case versionCommand.FullCommand():
		fmt.Println("0.1.0")

	case editCommand.FullCommand():
		cli.VerboseMsg("edit")
		readParams, err := openshift.NewParamsFromFile(*editFileArg, *privateKeyFlag)
		if err != nil {
			log.Fatalf("Could not read file: %s.", err.Error())
		}
		readContent, _ := readParams.Process(false, false)

		editedContent, err := cli.EditEnvFile(readContent)
		if err != nil {
			log.Fatalf("Could not edit file: %s.", err.Error())
		}
		editedParams := openshift.NewParamsFromInput(editedContent)

		renderedContent := editedParams.Render(*publicKeyDirFlag, readParams)
		err = ioutil.WriteFile(*editFileArg, []byte(renderedContent), 0644)
		if err != nil {
			log.Fatalf("Could not write file: %s.", err.Error())
		}

	case reEncryptCommand.FullCommand():
		cli.VerboseMsg("re-encrypt")
		for _, paramDir := range *paramDirFlag {
			files, err := ioutil.ReadDir(paramDir)
			if err != nil {
				log.Fatal(err)
			}
			filePattern := ".*\\.env"
			for _, file := range files {
				matched, _ := regexp.MatchString(filePattern, file.Name())
				if !matched {
					continue
				}
				filename := paramDir + string(os.PathSeparator) + file.Name()
				readParams, err := openshift.NewParamsFromFile(filename, *privateKeyFlag)
				if err != nil {
					log.Fatalf("Could not read file: %s.", err.Error())
				}
				readContent, _ := readParams.Process(false, false)

				editedParams := openshift.NewParamsFromInput(readContent)

				renderedContent := editedParams.Render(*publicKeyDirFlag, []*openshift.Param{})
				err = ioutil.WriteFile(filename, []byte(renderedContent), 0644)
				if err != nil {
					log.Fatalf("Could not write file: %s.", err.Error())
				}
			}
		}

	case revealCommand.FullCommand():
		cli.VerboseMsg("reveal")
		readParams, err := openshift.NewParamsFromFile(*revealFileArg, *privateKeyFlag)
		if err != nil {
			log.Fatalf("Could not read file: %s.", err.Error())
		}
		readContent, err := readParams.Process(false, true)
		if err != nil {
			log.Fatalf("Failed to process: %s.", err.Error())
		}
		fmt.Println(readContent)

	case statusCommand.FullCommand():
		cli.VerboseMsg("status")
		checkLoggedIn()

		updateRequired, _, err := calculateChangesets(
			*statusResourceArg,
			*selectorFlag,
			*templateDirFlag,
			*paramDirFlag,
			*statusLabelsFlag,
			*statusParamFlag,
			*statusParamFileFlag,
			*statusIgnoreUnknownParametersFlag,
			*statusUpsertOnlyFlag,
			*privateKeyFlag,
		)
		if err != nil {
			log.Fatalln(err.Error())
		}

		if updateRequired {
			os.Exit(3)
		}

	case exportCommand.FullCommand():
		cli.VerboseMsg("export")
		checkLoggedIn()

		filters, err := getFilters(*exportResourceArg, *selectorFlag)
		if err != nil {
			log.Fatalln(err.Error())
		}
		for _, f := range filters {
			export(f, *exportWriteFilesByKindFlag)
		}

	case updateCommand.FullCommand():
		cli.VerboseMsg("update")
		checkLoggedIn()

		updateRequired, changesets, err := calculateChangesets(
			*updateResourceArg,
			*selectorFlag,
			*templateDirFlag,
			*paramDirFlag,
			*updateLabelsFlag,
			*updateParamFlag,
			*updateParamFileFlag,
			*updateIgnoreUnknownParametersFlag,
			*updateUpsertOnlyFlag,
			*privateKeyFlag,
		)
		if err != nil {
			log.Fatalln(err.Error())
		}

		if updateRequired {
			if *nonInteractiveFlag {
				openshift.UpdateRemote(changesets)
			} else {
				c := cli.AskForConfirmation("Apply changes?")
				if c {
					openshift.UpdateRemote(changesets)
				}
			}
		}
	}
}

func calculateChangesets(resource string, selectorFlag string, templateDirs []string, paramDirs []string, label string, params []string, paramFile string, ignoreUnknownParameters bool, upsertOnly bool, privateKey string) (bool, map[string]*openshift.Changeset, error) {
	changesets := make(map[string]*openshift.Changeset)
	updateRequired := false

	filters, err := getFilters(resource, selectorFlag)
	if err != nil {
		return updateRequired, changesets, err
	}

	localResourceLists := assembleLocalResourceLists(
		filters,
		templateDirs,
		paramDirs,
		label,
		params,
		paramFile,
		ignoreUnknownParameters,
		privateKey,
	)
	remoteResourceLists := assembleRemoteResourceLists(filters)

	for k, _ := range filters {
		changesets[k] = compare(k, remoteResourceLists[k], localResourceLists[k], upsertOnly)
		if !changesets[k].Blank() {
			updateRequired = true
		}
	}
	return updateRequired, changesets, nil
}

// kindArgs might be blank, or a list of kinds (e.g. 'pvc,dc') or
// a kind/name combination (e.g. 'dc/foo').
// selectorFlag might be blank or a key and a label, e.g. 'name=foo'.
func getFilters(kindArg string, selectorFlag string) (map[string]*openshift.ResourceFilter, error) {
	filters := map[string]*openshift.ResourceFilter{}
	unknownKinds := []string{}
	targeted := make(map[string][]string)
	if len(kindArg) > 0 {
		kindArg = strings.ToLower(kindArg)
		kinds := strings.Split(kindArg, ",")
		for _, k := range kinds {
			kindParts := strings.Split(k, "/")

			// The first part is the kind, and potentially there is a
			// second part which is the name of one resource. It's okay if there
			// are duplicates in there as we use it only in an inclusion check
			// later on when we apply the filter.
			kind := kindParts[0]
			if _, ok := kindMapping[kind]; !ok {
				unknownKinds = append(unknownKinds, kind)
			} else {
				if len(kindParts) > 1 {
					if _, ok := targeted[kindMapping[kind]]; !ok {
						targeted[kindMapping[kind]] = []string{kindParts[1]}
					} else {
						targeted[kindMapping[kind]] = append(targeted[kindMapping[kind]], kindParts[1])
					}
				} else {
					if _, ok := targeted[kindMapping[kind]]; !ok {
						targeted[kindMapping[kind]] = []string{}
					}
				}

			}
		}
	} else {
		for _, v := range kindMapping {
			targeted[v] = []string{}
		}
	}

	// Abort if anything could not be read properly.
	if len(unknownKinds) > 0 {
		err := errors.New(fmt.Sprintf("Unknown resource kinds: %s", strings.Join(unknownKinds, ",")))
		return filters, err
	}

	for kind, names := range targeted {
		filter := &openshift.ResourceFilter{
			Kind:  kind,
			Names: names,
			Label: selectorFlag,
		}
		filters[kind] = filter
	}

	//cli.VerboseMsg("Selected kinds:", strings.Join(uniqueSelectedKinds, ","))

	return filters, nil
}

func checkLoggedIn() {
	cmd := cli.ExecPlainOcCmd([]string{"whoami"})
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalln("You need to login with 'oc login' first.")
	}
}

func assembleLocalResourceLists(filters map[string]*openshift.ResourceFilter, templateDirs []string, paramDirs []string, label string, params []string, paramFile string, ignoreUnknownParameters bool, privateKey string) map[string]*openshift.ResourceList {
	lists := initResourceLists(filters)

	// read files in folders and assemble lists for kinds
	for i, templateDir := range templateDirs {
		files, err := ioutil.ReadDir(templateDir)
		if err != nil {
			log.Fatal(err)
		}
		filePattern := ".*\\.ya?ml"
		for _, file := range files {
			matched, _ := regexp.MatchString(filePattern, file.Name())
			if !matched {
				continue
			}
			cli.VerboseMsg("Reading", file.Name())
			processedOut, err := openshift.ProcessTemplate(templateDir, file.Name(), paramDirs[i], label, params, paramFile, ignoreUnknownParameters, privateKey)
			if err != nil {
				log.Fatalln("Could not process", file.Name(), "template:", err.Error())
			}
			processedConfig := openshift.NewConfigFromList(processedOut)
			for _, l := range lists {
				l.AppendItems(processedConfig)
			}
		}
	}

	return lists
}

func assembleRemoteResourceLists(filters map[string]*openshift.ResourceFilter) map[string]*openshift.ResourceList {
	lists := initResourceLists(filters)

	// get kinds from remote and assemble lists
	for k, l := range lists {
		exportedOut, err := openshift.ExportResource(k)
		if err != nil {
			log.Fatalln("Could not export", k, " resources.")
		}
		exportedConfig := openshift.NewConfigFromList(exportedOut)
		l.AppendItems(exportedConfig)
	}

	return lists
}

func export(filter *openshift.ResourceFilter, writeFilesByKind bool) {
	out, err := openshift.ExportAsTemplate(filter)
	if err != nil {
		log.Fatalln("Could not export", filter.Kind, "resources as template.")
	}
	if len(out) == 0 {
		return
	}

	config := openshift.NewConfigFromTemplate(out)

	b, _ := yaml.Marshal(config.Processed)
	if writeFilesByKind {
		filename := kindToShortMapping[filter.Kind] + "-template.yml"
		ioutil.WriteFile(filename, b, 0644)
		fmt.Println("Exported", filter.Kind, "resources to", filename)
	} else {
		fmt.Println(string(b))
	}
}

func initResourceLists(filters map[string]*openshift.ResourceFilter) map[string]*openshift.ResourceList {
	lists := make(map[string]*openshift.ResourceList)
	for kind, filter := range filters {
		lists[kind] = &openshift.ResourceList{Filter: filter}
	}
	return lists
}

func compare(kind string, remoteResourceList *openshift.ResourceList, localResourceList *openshift.ResourceList, upsertOnly bool) *openshift.Changeset {
	fmt.Println("\n==========", kind, "resources", "==========")

	changeset := openshift.NewChangeset(remoteResourceList, localResourceList, upsertOnly)

	for itemName, _ := range changeset.Noop {
		fmt.Printf("* %s is in sync\n", itemName)
	}

	for itemName, itemConfigs := range changeset.Delete {
		cli.PrintRedf("- %s to be deleted\n", itemName)
		cli.ShowDiff(itemConfigs)
	}

	for itemName, itemConfigs := range changeset.Create {
		cli.PrintGreenf("+ %s to be created\n", itemName)
		cli.ShowDiff(itemConfigs)
	}

	for itemName, itemConfigs := range changeset.Update {
		cli.PrintYellowf("~ %s to be updated\n", itemName)
		cli.ShowDiff(itemConfigs)
	}

	return changeset
}
