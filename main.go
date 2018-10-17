package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/openshift"
	"github.com/opendevstack/tailor/utils"
)

var (
	app = kingpin.New(
		"tailor",
		"Tailor - Infrastructure as Code for OpenShift",
	).DefaultEnvars()
	verboseFlag = app.Flag(
		"verbose",
		"Enable verbose output.",
	).Short('v').Bool()
	debugFlag = app.Flag(
		"debug",
		"Enable debug output (implies verbose).",
	).Short('d').Bool()
	nonInteractiveFlag = app.Flag(
		"non-interactive",
		"Disable interactive mode.",
	).Bool()
	fileFlag = app.Flag(
		"file",
		"Tailorfile with flags.",
	).Short('f').Default("Tailorfile").String()

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
		"File(s) containing template parameter values to set/override in the template.",
	).Strings()
	statusDiffFlag = statusCommand.Flag(
		"diff",
		"Type of diff (text or json)",
	).Default("text").String()
	statusIgnorePathFlag = statusCommand.Flag(
		"ignore-path",
		"Path(s) per kind/name to ignore (e.g. because they are externally modified) in RFC 6901 format.",
	).PlaceHolder("bc:foobar:/spec/output/to/name").Strings()
	statusIgnoreUnknownParametersFlag = statusCommand.Flag(
		"ignore-unknown-parameters",
		"If true, will not stop processing if a provided parameter does not exist in the template.",
	).Bool()
	statusUpsertOnlyFlag = statusCommand.Flag(
		"upsert-only",
		"Don't delete resource, only create / update.",
	).Short('u').Bool()
	statusForceFlag = statusCommand.Flag(
		"force",
		"Force to delete all resources.",
	).Bool()
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
		"File(s) containing template parameter values to set/override in the template.",
	).Strings()
	updateDiffFlag = updateCommand.Flag(
		"diff",
		"Type of diff (text or json)",
	).Default("text").String()
	updateIgnorePathFlag = updateCommand.Flag(
		"ignore-path",
		"Path(s) per kind to ignore (e.g. because they are externally modified) in RFC 6901 format.",
	).PlaceHolder("bc:foobar:/spec/output/to/name").Strings()
	updateIgnoreUnknownParametersFlag = updateCommand.Flag(
		"ignore-unknown-parameters",
		"If true, will not stop processing if a provided parameter does not exist in the template.",
	).Bool()
	updateUpsertOnlyFlag = updateCommand.Flag(
		"upsert-only",
		"Don't delete resource, only create / update.",
	).Short('u').Bool()
	updateForceFlag = updateCommand.Flag(
		"force",
		"Force to delete all resources.",
	).Bool()
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
)

func main() {
	defer func() {
		err := recover()
		if err != nil {
			log.Fatalf("Fatal error: %s - %s.", err, debug.Stack())
		}
	}()

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	fileFlags, err := cli.GetFileFlags(*fileFlag, (*verboseFlag || *debugFlag))
	if err != nil {
		log.Fatalln("Could not read Tailorfile:", err)
	}
	globalOptions := &cli.GlobalOptions{}
	globalOptions.UpdateWithFile(fileFlags)
	globalOptions.UpdateWithFlags(
		*verboseFlag,
		*debugFlag,
		*nonInteractiveFlag,
		*namespaceFlag,
		*selectorFlag,
		*templateDirFlag,
		*paramDirFlag,
		*publicKeyDirFlag,
		*privateKeyFlag,
		*passphraseFlag,
	)
	err = globalOptions.Process()
	if err != nil {
		log.Fatalln("Options could not be processed:", err)
	}

	switch command {
	case versionCommand.FullCommand():
		fmt.Println("0.8.0")

	case editCommand.FullCommand():
		readParams, err := openshift.NewParamsFromFile(*editFileArg, globalOptions.PrivateKey, globalOptions.Passphrase)
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

		renderedContent, err := editedParams.Render(globalOptions.PublicKeyDir, readParams)
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(*editFileArg, []byte(renderedContent), 0644)
		if err != nil {
			log.Fatalf("Could not write file: %s.", err)
		}

	case reEncryptCommand.FullCommand():
		if len(*reEncryptFileArg) > 0 {
			err := reEncrypt(*reEncryptFileArg, globalOptions.PrivateKey, globalOptions.Passphrase, globalOptions.PublicKeyDir)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			for _, paramDir := range globalOptions.ParamDirs {
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
					err := reEncrypt(filename, globalOptions.PrivateKey, globalOptions.Passphrase, globalOptions.PublicKeyDir)
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
		readParams, err := openshift.NewParamsFromFile(*revealFileArg, globalOptions.PrivateKey, globalOptions.Passphrase)
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
		privateKeyFilename := globalOptions.PrivateKey
		utils.PrintPrivateKey(entity, privateKeyFilename)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("Private Key written to %s. This file MUST NOT be committed.\n", privateKeyFilename)

	case statusCommand.FullCommand():
		compareOptions := &cli.CompareOptions{
			GlobalOptions: globalOptions,
		}
		compareOptions.UpdateWithFile(fileFlags)
		compareOptions.UpdateWithFlags(
			*statusLabelsFlag,
			*statusParamFlag,
			*statusParamFileFlag,
			*statusDiffFlag,
			*statusIgnorePathFlag,
			*statusIgnoreUnknownParametersFlag,
			*statusUpsertOnlyFlag,
			*statusForceFlag,
			*statusResourceArg,
		)
		err := compareOptions.Process()
		if err != nil {
			log.Fatalln("Options could not be processed:", err)
		}

		updateRequired, _, err := calculateChangeset(compareOptions)
		if err != nil {
			log.Fatalln(err)
		}

		if updateRequired {
			os.Exit(3)
		}

	case updateCommand.FullCommand():
		compareOptions := &cli.CompareOptions{
			GlobalOptions: globalOptions,
		}
		compareOptions.UpdateWithFile(fileFlags)
		compareOptions.UpdateWithFlags(
			*updateLabelsFlag,
			*updateParamFlag,
			*updateParamFileFlag,
			*updateDiffFlag,
			*updateIgnorePathFlag,
			*updateIgnoreUnknownParametersFlag,
			*updateUpsertOnlyFlag,
			*updateForceFlag,
			*updateResourceArg,
		)
		err := compareOptions.Process()
		if err != nil {
			log.Fatalln("Options could not be processed:", err)
		}

		updateRequired, changeset, err := calculateChangeset(compareOptions)
		if err != nil {
			log.Fatalln(err)
		}

		if updateRequired {
			if globalOptions.NonInteractive {
				err = changeset.Apply(compareOptions)
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				c := cli.AskForConfirmation("Apply changes?")
				if c {
					err = changeset.Apply(compareOptions)
					if err != nil {
						log.Fatalln(err)
					}
				}
			}
		}

	case exportCommand.FullCommand():
		exportOptions := &cli.ExportOptions{
			GlobalOptions: globalOptions,
		}
		exportOptions.UpdateWithFile(fileFlags)
		exportOptions.UpdateWithFlags(
			*exportResourceArg,
		)
		err := exportOptions.Process()
		if err != nil {
			log.Fatalln("Options could not be processed:", err)
		}

		filter, err := getFilter(exportOptions.Resource, exportOptions.Selector)
		if err != nil {
			log.Fatalln(err)
		}
		export(filter, exportOptions)
	}
}

func reEncrypt(filename, privateKey, passphrase, publicKeyDir string) error {
	readParams, err := openshift.NewParamsFromFile(filename, privateKey, passphrase)
	if err != nil {
		return fmt.Errorf("Could not read file: %s", err)
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
		return fmt.Errorf("Could not write file: %s", err)
	}
	return nil
}

func calculateChangeset(compareOptions *cli.CompareOptions) (bool, *openshift.Changeset, error) {
	updateRequired := false

	where := strings.Join(compareOptions.TemplateDirs, ", ")
	if len(compareOptions.TemplateDirs) == 1 && compareOptions.TemplateDirs[0] == "." {
		where, _ = os.Getwd()
	}

	fmt.Printf(
		"Comparing templates in %s with OCP namespace %s.\n",
		where,
		compareOptions.Namespace,
	)

	if len(compareOptions.Resource) > 0 && len(compareOptions.Selector) > 0 {
		fmt.Printf(
			"Limiting resources to %s with selector %s.\n",
			compareOptions.Resource,
			compareOptions.Selector,
		)
	} else if len(compareOptions.Selector) > 0 {
		fmt.Printf(
			"Limiting to resources with selector %s.\n",
			compareOptions.Selector,
		)
	} else if len(compareOptions.Resource) > 0 {
		fmt.Printf(
			"Limiting resources to %s.\n",
			compareOptions.Resource,
		)
	}

	resource := compareOptions.Resource
	selectorFlag := compareOptions.Selector

	filter, err := getFilter(resource, selectorFlag)
	if err != nil {
		return updateRequired, &openshift.Changeset{}, err
	}

	templateBasedList := assembleTemplateBasedList(
		filter,
		compareOptions,
	)
	platformBasedList := assemblePlatformBasedList(filter, compareOptions)
	platformResourcesWord := "resources"
	if platformBasedList.Length() == 1 {
		platformResourcesWord = "resource"
	}
	templateResourcesWord := "resources"
	if templateBasedList.Length() == 1 {
		templateResourcesWord = "resource"
	}
	fmt.Printf(
		"Found %d %s in OCP cluster (current state) and %d %s in processed templates (desired state).\n\n",
		platformBasedList.Length(),
		platformResourcesWord,
		templateBasedList.Length(),
		templateResourcesWord,
	)

	if templateBasedList.Length() == 0 && !compareOptions.Force {
		fmt.Printf("No items where found in desired state. ")
		if len(compareOptions.Resource) == 0 && len(compareOptions.Selector) == 0 {
			fmt.Printf(
				"Are there any templates in %s?\n",
				where,
			)
		} else {
			fmt.Printf(
				"Possible reasons are:\n"+
					"* No templates are located in %s\n",
				where,
			)
			if len(compareOptions.Resource) > 0 {
				fmt.Printf(
					"* No templates contain resources of kinds: %s\n",
					compareOptions.Resource,
				)
			}
			if len(compareOptions.Selector) > 0 {
				fmt.Printf(
					"* No templates contain resources matching selector: %s\n",
					compareOptions.Selector,
				)
			}
		}
		fmt.Println("\nRefusing to continue without --force")
		return updateRequired, &openshift.Changeset{}, nil
	}

	changeset, err := compare(
		platformBasedList,
		templateBasedList,
		compareOptions.UpsertOnly,
		compareOptions.Diff,
		compareOptions.IgnorePaths,
	)
	if err != nil {
		return false, changeset, err
	}
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
		filter.Name = openshift.KindMapping[nameParts[0]] + "/" + nameParts[1]
		return filter, nil
	}

	targetedKinds := make(map[string]bool)
	unknownKinds := []string{}
	kinds := strings.Split(kindArg, ",")
	for _, kind := range kinds {
		if _, ok := openshift.KindMapping[kind]; !ok {
			unknownKinds = append(unknownKinds, kind)
		} else {
			targetedKinds[openshift.KindMapping[kind]] = true
		}
	}

	if len(unknownKinds) > 0 {
		return nil, fmt.Errorf(
			"Unknown resource kinds: %s",
			strings.Join(unknownKinds, ","),
		)
	}

	for kind := range targetedKinds {
		filter.Kinds = append(filter.Kinds, kind)
	}

	sort.Strings(filter.Kinds)

	return filter, nil
}

func assembleTemplateBasedList(filter *openshift.ResourceFilter, compareOptions *cli.CompareOptions) *openshift.ResourceList {
	list := &openshift.ResourceList{Filter: filter}

	// read files in folders and assemble lists for kinds
	for i, templateDir := range compareOptions.TemplateDirs {
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
			cli.DebugMsg("Reading template", file.Name())
			processedOut, err := openshift.ProcessTemplate(templateDir, file.Name(), compareOptions.ParamDirs[i], compareOptions)
			if err != nil {
				log.Fatalln("Could not process", file.Name(), "template:", err)
			}
			list.CollectItemsFromTemplateList(processedOut)
		}
	}

	return list
}

func assemblePlatformBasedList(filter *openshift.ResourceFilter, compareOptions *cli.CompareOptions) *openshift.ResourceList {
	list := &openshift.ResourceList{Filter: filter}

	exportedOut, err := openshift.ExportResources(filter, compareOptions)
	if err != nil {
		log.Fatalln("Could not export", filter.String(), " resources.")
	}
	list.CollectItemsFromPlatformList(exportedOut)

	return list
}

func export(filter *openshift.ResourceFilter, exportOptions *cli.ExportOptions) {
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

	out, err := openshift.ExportAsTemplate(filter, templateName, exportOptions)
	if err != nil {
		log.Fatalln(
			"Could not export",
			filter.String(),
			"resources as template:",
			err,
		)
	}

	fmt.Println(out)
}

func compare(remoteResourceList *openshift.ResourceList, localResourceList *openshift.ResourceList, upsertOnly bool, diff string, ignorePaths []string) (*openshift.Changeset, error) {
	changeset, err := openshift.NewChangeset(remoteResourceList, localResourceList, upsertOnly, ignorePaths)
	if err != nil {
		return changeset, err
	}

	for _, change := range changeset.Noop {
		fmt.Printf("* %s is in sync\n", change.ItemName())
	}

	for _, change := range changeset.Delete {
		cli.PrintRedf("- %s to delete\n", change.ItemName())
		fmt.Printf(change.Diff())
	}

	for _, change := range changeset.Create {
		cli.PrintGreenf("+ %s to create\n", change.ItemName())
		fmt.Printf(change.Diff())
	}

	for _, change := range changeset.Update {
		cli.PrintYellowf("~ %s to update\n", change.ItemName())
		if diff == "text" {
			fmt.Printf(change.Diff())
		} else {
			fmt.Println(change.JsonPatches(true))
		}
	}

	if !changeset.Blank() {
		fmt.Printf("\nChange Summary: ")
		cli.PrintGreenf("%d to create", len(changeset.Create))
		fmt.Printf(", ")
		cli.PrintYellowf("%d to update", len(changeset.Update))
		fmt.Printf(", ")
		cli.PrintRedf("%d to delete\n", len(changeset.Delete))
	}

	return changeset, nil
}
