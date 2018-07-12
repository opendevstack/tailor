package main

import (
	"errors"
	"fmt"
	"github.com/alecthomas/kingpin"
	"github.com/ghodss/yaml"
	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/openshift"
	"github.com/opendevstack/tailor/utils"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
)

var (
	app = kingpin.New(
		"tailor",
		"OC Diff Tool",
	).DefaultEnvars().UsageTemplate(kingpin.LongHelpTemplate)
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
	passphraseFlag = app.Flag(
		"passphrase",
		"Passphrase to unlock key",
	).String()

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
	).Required().String()

	reEncryptCommand = secretsCommand.Command(
		"re-encrypt",
		"Re-Encrypt param file(s)",
	)
	reEncryptFileArg = reEncryptCommand.Arg(
		"file", "File to re-encrypt",
	).String()

	revealCommand = secretsCommand.Command(
		"reveal",
		"Show param file contents with revealed secrets",
	)
	revealFileArg = revealCommand.Arg(
		"file", "File to show",
	).Required().String()

	generateKeyCommand = secretsCommand.Command(
		"generate-key",
		"Generate new keypair",
	)
	generateKeyNameFlag = generateKeyCommand.Flag(
		"name",
		"Name for keypair",
	).String()
	generateKeyEmailArg = generateKeyCommand.Arg(
		"email", "Emil of keypair",
	).Required().String()

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
)

func main() {
	defer func() {
		err := recover()
		log.Fatalf("Fatal error: %s - %s.", err, debug.Stack())
	}()

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	cli.SetOptions(*verboseFlag, *namespaceFlag, *selectorFlag)

	paramDir := *paramDirFlag
	if (len(paramDir) > 1 || paramDir[0] != ".") && (len(*statusParamFileFlag) > 0 || len(*updateParamFileFlag) > 0) {
		log.Fatalln("You cannot specify both --param-dir and --param-file.")
	}

	switch command {
	case versionCommand.FullCommand():
		fmt.Println("0.4.0")

	case editCommand.FullCommand():
		readParams, err := openshift.NewParamsFromFile(*editFileArg, *privateKeyFlag, *passphraseFlag)
		if err != nil {
			log.Fatalf("Could not read file: %s.", err)
		}
		readContent, _ := readParams.Process(false, false)

		editedContent, err := cli.EditEnvFile(readContent)
		if err != nil {
			log.Fatalf("Could not edit file: %s.", err)
		}
		editedParams, err := openshift.NewParamsFromInput(editedContent)
		if err != nil {
			log.Fatal(err)
		}

		renderedContent, err := editedParams.Render(*publicKeyDirFlag, readParams)
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(*editFileArg, []byte(renderedContent), 0644)
		if err != nil {
			log.Fatalf("Could not write file: %s.", err)
		}

	case reEncryptCommand.FullCommand():
		if len(*reEncryptFileArg) > 0 {
			err := reEncrypt(*reEncryptFileArg, *privateKeyFlag, *passphraseFlag, *publicKeyDirFlag)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			for _, paramDir := range *paramDirFlag {
				files, err := ioutil.ReadDir(paramDir)
				if err != nil {
					log.Fatal(err)
				}
				filePattern := ".*\\.env$"
				for _, file := range files {
					matched, _ := regexp.MatchString(filePattern, file.Name())
					if !matched {
						continue
					}
					filename := paramDir + string(os.PathSeparator) + file.Name()
					err := reEncrypt(filename, *privateKeyFlag, *passphraseFlag, *publicKeyDirFlag)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}

	case revealCommand.FullCommand():
		if _, err := os.Stat(*revealFileArg); os.IsNotExist(err) {
			log.Fatalf("'%s' does not exist.", *revealFileArg)
		}
		readParams, err := openshift.NewParamsFromFile(*revealFileArg, *privateKeyFlag, *passphraseFlag)
		if err != nil {
			log.Fatalf("Could not read file: %s.", err)
		}
		readContent, err := readParams.Process(false, true)
		if err != nil {
			log.Fatalf("Failed to process: %s.", err)
		}
		fmt.Println(readContent)

	case generateKeyCommand.FullCommand():
		emailParts := strings.Split(*generateKeyEmailArg, "@")
		name := *generateKeyNameFlag
		if len(name) == 0 {
			name = emailParts[0]
		}
		entity, err := utils.CreateEntity(name, *generateKeyEmailArg)
		if err != nil {
			log.Fatalf("Failed to generate keypair: %s.", err)
		}
		publicKeyFilename := strings.Replace(emailParts[0], ".", "-", -1) + ".key"
		utils.PrintPublicKey(entity, publicKeyFilename)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("Public Key written to %s. This file can be committed.\n", publicKeyFilename)
		privateKeyFilename := *privateKeyFlag
		utils.PrintPrivateKey(entity, privateKeyFilename)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("Private Key written to %s. This file MUST NOT be committed.\n", privateKeyFilename)

	case statusCommand.FullCommand():
		checkLoggedIn()

		updateRequired, _, err := calculateChangeset(
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
			*passphraseFlag,
		)
		if err != nil {
			log.Fatalln(err)
		}

		if updateRequired {
			os.Exit(3)
		}

	case exportCommand.FullCommand():
		checkLoggedIn()

		filter, err := getFilter(*exportResourceArg, *selectorFlag)
		if err != nil {
			log.Fatalln(err)
		}
		export(filter)

	case updateCommand.FullCommand():
		checkLoggedIn()

		updateRequired, changeset, err := calculateChangeset(
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
			*passphraseFlag,
		)
		if err != nil {
			log.Fatalln(err)
		}

		if updateRequired {
			if *nonInteractiveFlag {
				openshift.UpdateRemote(changeset)
			} else {
				c := cli.AskForConfirmation("Apply changes?")
				if c {
					openshift.UpdateRemote(changeset)
				}
			}
		}
	}
}

func reEncrypt(filename, privateKey, passphrase, publicKeyDir string) error {
	readParams, err := openshift.NewParamsFromFile(filename, privateKey, passphrase)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not read file: %s.", err))
	}
	readContent, _ := readParams.Process(false, false)

	editedParams, err := openshift.NewParamsFromInput(readContent)
	if err != nil {
		return err
	}

	renderedContent, err := editedParams.Render(publicKeyDir, []*openshift.Param{})
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, []byte(renderedContent), 0644)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not write file: %s.", err))
	}
	return nil
}

func calculateChangeset(resource string, selectorFlag string, templateDirs []string, paramDirs []string, label string, params []string, paramFile string, ignoreUnknownParameters bool, upsertOnly bool, privateKey string, passphrase string) (bool, *openshift.Changeset, error) {
	updateRequired := false
	filter, err := getFilter(resource, selectorFlag)
	if err != nil {
		return updateRequired, &openshift.Changeset{}, err
	}

	localResourceList := assembleLocalResourceList(
		filter,
		templateDirs,
		paramDirs,
		label,
		params,
		paramFile,
		ignoreUnknownParameters,
		privateKey,
		passphrase,
	)
	remoteResourceList := assembleRemoteResourceList(filter)

	changeset := compare(remoteResourceList, localResourceList, upsertOnly)
	updateRequired = !changeset.Blank()
	return updateRequired, changeset, nil
}

// kindArgs might be blank, or a list of kinds (e.g. 'pvc,dc') or
// a kind/name combination (e.g. 'dc/foo').
// selectorFlag might be blank or a key and a label, e.g. 'name=foo'.
func getFilter(kindArg string, selectorFlag string) (*openshift.ResourceFilter, error) {
	filter := &openshift.ResourceFilter{
		Kinds: []string{},
		Name:  "",
		Label: selectorFlag,
	}

	if len(kindArg) == 0 {
		return filter, nil
	}

	kindArg = strings.ToLower(kindArg)

	if strings.Contains(kindArg, "/") {
		if strings.Contains(kindArg, ",") {
			return nil, errors.New(
				"You cannot target more than one resource name",
			)
		}
		nameParts := strings.Split(kindArg, "/")
		filter.Name = kindMapping[nameParts[0]] + "/" + nameParts[1]
		return filter, nil
	}

	targetedKinds := make(map[string]bool)
	unknownKinds := []string{}
	kinds := strings.Split(kindArg, ",")
	for _, kind := range kinds {
		if _, ok := kindMapping[kind]; !ok {
			unknownKinds = append(unknownKinds, kind)
		} else {
			targetedKinds[kindMapping[kind]] = true
		}
	}

	if len(unknownKinds) > 0 {
		return nil, errors.New(fmt.Sprintf(
			"Unknown resource kinds: %s",
			strings.Join(unknownKinds, ","),
		))
	}

	for kind, _ := range targetedKinds {
		filter.Kinds = append(filter.Kinds, kind)
	}

	sort.Strings(filter.Kinds)

	return filter, nil
}

func checkLoggedIn() {
	cmd := cli.ExecPlainOcCmd([]string{"whoami"})
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalln("You need to login with 'oc login' first.")
	}
}

func assembleLocalResourceList(filter *openshift.ResourceFilter, templateDirs []string, paramDirs []string, label string, params []string, paramFile string, ignoreUnknownParameters bool, privateKey string, passphrase string) *openshift.ResourceList {
	list := &openshift.ResourceList{Filter: filter}

	// read files in folders and assemble lists for kinds
	for i, templateDir := range templateDirs {
		files, err := ioutil.ReadDir(templateDir)
		if err != nil {
			log.Fatal(err)
		}
		filePattern := ".*\\.ya?ml$"
		for _, file := range files {
			matched, _ := regexp.MatchString(filePattern, file.Name())
			if !matched {
				continue
			}
			cli.VerboseMsg("Reading", file.Name())
			processedOut, err := openshift.ProcessTemplate(templateDir, file.Name(), paramDirs[i], label, params, paramFile, ignoreUnknownParameters, privateKey, passphrase)
			if err != nil {
				log.Fatalln("Could not process", file.Name(), "template:", err)
			}
			processedConfig := openshift.NewConfigFromList(processedOut)
			list.AppendItems(processedConfig)
		}
	}

	return list
}

func assembleRemoteResourceList(filter *openshift.ResourceFilter) *openshift.ResourceList {
	list := &openshift.ResourceList{Filter: filter}

	exportedOut, err := openshift.ExportResources(filter)
	if err != nil {
		log.Fatalln("Could not export", filter.String(), " resources.")
	}
	exportedConfig := openshift.NewConfigFromList(exportedOut)
	list.AppendItems(exportedConfig)

	return list
}

func export(filter *openshift.ResourceFilter) {
	var templateName string
	if len(filter.Name) > 0 {
		templateName = strings.Replace(filter.Name, "/", "-", -1)
	} else if len(filter.Label) > 0 {
		labelParts := strings.Split(filter.Label, "=")
		templateName = labelParts[1]
	} else if len(filter.Kinds) > 0 {
		templateName = strings.ToLower(strings.Join(filter.Kinds, "-"))
	} else {
		templateName = "all"
	}

	out, err := openshift.ExportAsTemplate(filter, templateName)
	if err != nil {
		log.Fatalln("Could not export", filter.String(), "resources as template.")
	}
	if len(out) == 0 {
		return
	}

	config := openshift.NewConfigFromTemplate(out)

	b, _ := yaml.Marshal(config.Processed)
	fmt.Println(string(b))
}

func compare(remoteResourceList *openshift.ResourceList, localResourceList *openshift.ResourceList, upsertOnly bool) *openshift.Changeset {
	changeset := openshift.NewChangeset(remoteResourceList, localResourceList, upsertOnly)

	for _, change := range changeset.Noop {
		fmt.Printf("* %s is in sync\n", change.ItemName())
	}

	for _, change := range changeset.Delete {
		cli.PrintRedf("- %s to be deleted\n", change.ItemName())
		cli.ShowDiff(change.CurrentState, change.DesiredState)
	}

	for _, change := range changeset.Create {
		cli.PrintGreenf("+ %s to be created\n", change.ItemName())
		cli.ShowDiff(change.CurrentState, change.DesiredState)
	}

	for _, change := range changeset.Update {
		cli.PrintYellowf("~ %s to be updated\n", change.ItemName())
		cli.ShowDiff(change.CurrentState, change.DesiredState)
	}

	return changeset
}
