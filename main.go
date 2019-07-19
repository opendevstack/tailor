package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kingpin"
	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/commands"
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
	ocBinaryFlag = app.Flag(
		"oc-binary",
		"oc binary to use",
	).Default("oc").String()
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
	excludeFlag = app.Flag(
		"exclude",
		"Exclude kinds, names and labels (comma separated)",
	).Short('e').String()
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
	forceFlag = app.Flag(
		"force",
		"Force to continue despite warning (e.g. deleting all resources).",
	).Bool()

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

	if command == versionCommand.FullCommand() {
		fmt.Println("0.9.4+master")
		return
	}

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
		*ocBinaryFlag,
		*namespaceFlag,
		*selectorFlag,
		*excludeFlag,
		*templateDirFlag,
		*paramDirFlag,
		*publicKeyDirFlag,
		*privateKeyFlag,
		*passphraseFlag,
		*forceFlag,
	)
	err = globalOptions.Process()
	if err != nil {
		log.Fatalln("Options could not be processed:", err)
	}

	switch command {
	case editCommand.FullCommand():
		err := commands.Edit(globalOptions, *editFileArg)
		if err != nil {
			log.Fatalf("Failed to edit file: %s.", err)
		}

	case reEncryptCommand.FullCommand():
		err := commands.ReEncrypt(globalOptions, *reEncryptFileArg)
		if err != nil {
			log.Fatalf("Failed to re-encrypt: %s.", err)
		}

	case revealCommand.FullCommand():
		err := commands.Reveal(globalOptions, *revealFileArg)
		if err != nil {
			log.Fatalf("Failed to reveal file: %s.", err)
		}

	case generateKeyCommand.FullCommand():
		err := commands.GenerateKey(globalOptions, *generateKeyEmailArg, *generateKeyNameFlag)
		if err != nil {
			log.Fatalf("Failed to generate keypair: %s.", err)
		}

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
			*statusResourceArg,
		)
		err := compareOptions.Process()
		if err != nil {
			log.Fatalln("Options could not be processed:", err)
		}

		updateRequired, _, err := commands.Status(compareOptions)
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
			*updateResourceArg,
		)
		err := compareOptions.Process()
		if err != nil {
			log.Fatalln("Options could not be processed:", err)
		}

		err = commands.Update(compareOptions)
		if err != nil {
			log.Fatalln(err)
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
		err = commands.Export(exportOptions)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
