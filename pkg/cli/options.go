package cli

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/opendevstack/tailor/pkg/utils"
)

// GlobalOptions are app-wide that cannot be modified within a context.
type GlobalOptions struct {
	Verbose        bool
	Debug          bool
	NonInteractive bool
	OcBinary       string
	File           string
	ContextDirs    []string
	Force          bool
	IsLoggedIn     bool
}

// NamespaceOptions are context-wide.
type NamespaceOptions struct {
	Namespace         string
	CheckedNamespaces []string
}

// CompareOptions are context-wide.
type CompareOptions struct {
	*GlobalOptions
	*NamespaceOptions
	ContextDir              string
	Selector                string
	Exclude                 string
	TemplateDir             string
	ParamDir                string
	PrivateKey              string
	Passphrase              string
	Labels                  string
	Params                  []string
	ParamFiles              []string
	Format                  string
	PreservePaths           []string
	PreserveImmutableFields bool
	IgnoreUnknownParameters bool
	UpsertOnly              bool
	AllowRecreate           bool
	RevealSecrets           bool
	Resource                string
}

// ExportOptions are context-wide.
type ExportOptions struct {
	*GlobalOptions
	*NamespaceOptions
	ContextDir      string
	Selector        string
	Exclude         string
	TemplateDir     string
	ParamDir        string
	WithAnnotations bool
	Resource        string
}

// SecretsOptions are context-wide.
type SecretsOptions struct {
	*GlobalOptions
	ContextDir   string
	ParamDir     string
	PublicKeyDir string
	PrivateKey   string
	Passphrase   string
}

// NewGlobalOptions returns new global options based on file/flags.
// Those options are shared across all commands.
func NewGlobalOptions(
	clusterRequired bool,
	fileFlag string,
	verboseFlag bool,
	debugFlag bool,
	nonInteractiveFlag bool,
	ocBinaryFlag string,
	contextDirFlag []string,
	forceFlag bool) (*GlobalOptions, error) {
	o := &GlobalOptions{}

	fileFlags, err := getFileFlags(fileFlag, verbose)
	if err != nil {
		return o, fmt.Errorf("Could not read %s: %s", fileFlag, err)
	}

	if verboseFlag {
		o.Verbose = true
	} else if fileFlags["verbose"] == "true" {
		o.Verbose = true
	}

	if debugFlag {
		o.Debug = true
	} else if fileFlags["debug"] == "true" {
		o.Debug = true
	}

	if nonInteractiveFlag {
		o.NonInteractive = true
	} else if fileFlags["non-interactive"] == "true" {
		o.NonInteractive = true
	}

	if len(fileFlag) > 0 {
		o.File = fileFlag
	}

	if len(ocBinaryFlag) > 0 {
		o.OcBinary = ocBinaryFlag
	} else if val, ok := fileFlags["oc-binary"]; ok {
		o.OcBinary = val
	}

	if forceFlag {
		o.Force = true
	} else if fileFlags["force"] == "true" {
		o.Force = true
	}

	o.ContextDirs = contextDirFlag
	if len(contextDirFlag) > 1 || contextDirFlag[0] != "." {
		o.ContextDirs = contextDirFlag
	} else if val, ok := fileFlags["context-dir"]; ok {
		o.ContextDirs = strings.Split(val, ",")
	}

	verbose = o.Verbose || o.Debug
	debug = o.Debug
	ocBinary = o.OcBinary

	DebugMsg(fmt.Sprintf("%#v", o))

	return o, o.check(clusterRequired)
}

// NewCompareOptions returns new options for the diff/apply command based on file/flags.
func NewCompareOptions(
	globalOptions *GlobalOptions,
	contextDir string,
	namespaceFlag string,
	selectorFlag string,
	excludeFlag string,
	templateDirFlag string,
	paramDirFlag string,
	publicKeyDirFlag string,
	privateKeyFlag string,
	passphraseFlag string,
	labelsFlag string,
	paramFlag []string,
	paramFileFlag []string,
	formatFlag string,
	preserveFlag []string,
	preserveImmutableFieldsFlag bool,
	ignoreUnknownParametersFlag bool,
	upsertOnlyFlag bool,
	allowRecreateFlag bool,
	revealSecretsFlag bool,
	resourceArg string) (*CompareOptions, error) {
	o := &CompareOptions{
		GlobalOptions:    globalOptions,
		NamespaceOptions: &NamespaceOptions{},
		ContextDir:       contextDir,
	}
	filename := utils.AbsoluteOrRelativePath(o.File, contextDir)

	fileFlags, err := getFileFlags(filename, verbose)
	if err != nil {
		return o, fmt.Errorf("Could not read %s: %s", filename, err)
	}

	if len(namespaceFlag) > 0 {
		o.Namespace = namespaceFlag
	} else if val, ok := fileFlags["namespace"]; ok {
		o.Namespace = val
	}

	if len(selectorFlag) > 0 {
		o.Selector = selectorFlag
	} else if val, ok := fileFlags["selector"]; ok {
		o.Selector = val
	}

	if len(excludeFlag) > 0 {
		o.Exclude = excludeFlag
	} else if val, ok := fileFlags["exclude"]; ok {
		o.Exclude = val
	}

	o.TemplateDir = "."
	if templateDirFlag != "." {
		o.TemplateDir = templateDirFlag
	} else if val, ok := fileFlags["template-dir"]; ok {
		o.TemplateDir = val
	}

	o.ParamDir = "."
	if paramDirFlag != "." {
		o.ParamDir = paramDirFlag
	} else if val, ok := fileFlags["param-dir"]; ok {
		o.ParamDir = val
	}

	o.PrivateKey = "private.key"
	if privateKeyFlag != "private.key" {
		o.PrivateKey = privateKeyFlag
	} else if val, ok := fileFlags["private-key"]; ok {
		o.PrivateKey = val
	}

	if len(passphraseFlag) > 0 {
		o.Passphrase = passphraseFlag
	} else if val, ok := fileFlags["passphrase"]; ok {
		o.Passphrase = val
	}

	if len(labelsFlag) > 0 {
		o.Labels = labelsFlag
	} else if val, ok := fileFlags["labels"]; ok {
		o.Labels = val
	}

	if val, ok := fileFlags["param"]; ok {
		o.Params = strings.Split(val, ",")
	}
	if len(paramFlag) > 0 {
		params := map[string]string{}
		for _, setParam := range o.Params {
			setPair := strings.SplitN(setParam, "=", 2)
			key := setPair[0]
			params[key] = setPair[1]
			for _, newParam := range paramFlag {
				newPair := strings.SplitN(newParam, "=", 2)
				if key == newPair[0] {
					params[key] = newPair[1]
					break
				}
			}
		}
		o.Params = []string{}
		for k, v := range params {
			o.Params = append(o.Params, k+"="+v)
		}
		for _, v := range paramFlag {
			pair := strings.SplitN(v, "=", 2)
			if _, ok := params[pair[0]]; !ok {
				o.Params = append(o.Params, v)
			}
		}
	}

	if len(paramFileFlag) > 0 {
		o.ParamFiles = paramFileFlag
	} else if val, ok := fileFlags["param-file"]; ok {
		o.ParamFiles = strings.Split(val, ",")
	}

	if len(formatFlag) > 0 {
		o.Format = formatFlag
	} else if val, ok := fileFlags["diff"]; ok {
		o.Format = val
	}

	if len(preserveFlag) > 0 {
		o.PreservePaths = preserveFlag
	} else if val, ok := fileFlags["preserve"]; ok {
		o.PreservePaths = strings.Split(val, ",")
	}

	if preserveImmutableFieldsFlag {
		o.PreserveImmutableFields = true
	} else if fileFlags["preserve-immutable-fields"] == "true" {
		o.PreserveImmutableFields = true
	}

	if ignoreUnknownParametersFlag {
		o.IgnoreUnknownParameters = true
	} else if fileFlags["ignore-unknown-parameters"] == "true" {
		o.IgnoreUnknownParameters = true
	}

	if upsertOnlyFlag {
		o.UpsertOnly = true
	} else if fileFlags["upsert-only"] == "true" {
		o.UpsertOnly = true
	}

	if allowRecreateFlag {
		o.AllowRecreate = true
	} else if fileFlags["allow-recreate"] == "true" {
		o.AllowRecreate = true
	}

	if revealSecretsFlag {
		o.RevealSecrets = true
	} else if fileFlags["reveal-secrets"] == "true" {
		o.RevealSecrets = true
	}

	if len(resourceArg) > 0 {
		o.Resource = resourceArg
	} else if val, ok := fileFlags["resource"]; ok {
		o.Resource = val
	}

	DebugMsg(fmt.Sprintf("%#v", o))

	return o, o.check()
}

// NewExportOptions returns new options for the export command based on file/flags.
func NewExportOptions(
	globalOptions *GlobalOptions,
	contextDir string,
	namespaceFlag string,
	selectorFlag string,
	excludeFlag string,
	templateDirFlag string,
	paramDirFlag string,
	withAnnotationsFlag bool,
	resourceArg string) (*ExportOptions, error) {
	o := &ExportOptions{
		GlobalOptions:    globalOptions,
		NamespaceOptions: &NamespaceOptions{},
		ContextDir:       contextDir,
	}
	filename := utils.AbsoluteOrRelativePath(o.File, contextDir)

	fileFlags, err := getFileFlags(filename, verbose)
	if err != nil {
		return o, fmt.Errorf("Could not read %s: %s", filename, err)
	}

	if len(namespaceFlag) > 0 {
		o.Namespace = namespaceFlag
	} else if val, ok := fileFlags["namespace"]; ok {
		o.Namespace = val
	}

	if len(selectorFlag) > 0 {
		o.Selector = selectorFlag
	} else if val, ok := fileFlags["selector"]; ok {
		o.Selector = val
	}

	if len(excludeFlag) > 0 {
		o.Exclude = excludeFlag
	} else if val, ok := fileFlags["exclude"]; ok {
		o.Exclude = val
	}

	o.TemplateDir = "."
	if templateDirFlag != "." {
		o.TemplateDir = templateDirFlag
	} else if val, ok := fileFlags["template-dir"]; ok {
		o.TemplateDir = val
	}

	o.ParamDir = "."
	if paramDirFlag != "." {
		o.ParamDir = paramDirFlag
	} else if val, ok := fileFlags["param-dir"]; ok {
		o.ParamDir = val
	}

	if withAnnotationsFlag {
		o.WithAnnotations = true
	} else if fileFlags["with-annotations"] == "true" {
		o.WithAnnotations = true
	}

	if len(resourceArg) > 0 {
		o.Resource = resourceArg
	} else if val, ok := fileFlags["resource"]; ok {
		o.Resource = val
	}

	DebugMsg(fmt.Sprintf("%#v", o))

	return o, o.check()
}

// NewSecretsOptions returns new options for the secrets subcommand based on file/flags.
func NewSecretsOptions(
	globalOptions *GlobalOptions,
	paramDirFlag string,
	publicKeyDirFlag string,
	privateKeyFlag string,
	passphraseFlag string) (*SecretsOptions, error) {
	contextDir := globalOptions.ContextDirs[0]
	o := &SecretsOptions{
		GlobalOptions: globalOptions,
		ContextDir:    contextDir,
	}
	filename := utils.AbsoluteOrRelativePath(o.File, contextDir)

	fileFlags, err := getFileFlags(filename, verbose)
	if err != nil {
		return o, fmt.Errorf("Could not read %s: %s", filename, err)
	}

	o.ParamDir = "."
	if paramDirFlag != "." {
		o.ParamDir = paramDirFlag
	} else if val, ok := fileFlags["param-dir"]; ok {
		o.ParamDir = val
	}

	o.PublicKeyDir = "."
	if publicKeyDirFlag != "." {
		o.PublicKeyDir = publicKeyDirFlag
	} else if val, ok := fileFlags["public-key-dir"]; ok {
		o.PublicKeyDir = val
	}

	o.PrivateKey = "private.key"
	if privateKeyFlag != "private.key" {
		o.PrivateKey = privateKeyFlag
	} else if val, ok := fileFlags["private-key"]; ok {
		o.PrivateKey = val
	}

	DebugMsg(fmt.Sprintf("%#v", o))

	return o, o.check()
}

func (o *GlobalOptions) check(clusterRequired bool) error {
	if !o.checkOcBinary() {
		return fmt.Errorf("No such oc binary: %s", o.OcBinary)
	}
	if clusterRequired {
		if !o.checkLoggedIn() {
			return errors.New("You need to login with 'oc login' first")
		}
		c := NewOcClient("")
		if v := ocVersion(c); !v.Matches() {
			errorMsg := fmt.Sprintf("Version mismatch between client (%s) and server (%s) detected. "+
				"This can lead to incorrect behaviour. "+
				"Update your oc binary or point to an alternative binary with --oc-binary.", v.client, v.server)
			if !o.Force {
				return fmt.Errorf("%s\n\nRefusing to continue without --force", errorMsg)
			}
			VerboseMsg(errorMsg)
		}
	}
	return nil
}

func (o *GlobalOptions) checkLoggedIn() bool {
	if !o.IsLoggedIn {
		c := NewOcClient("")
		loggedIn, err := c.CheckLoggedIn()
		if err != nil {
			VerboseMsg(err.Error())
		}
		o.IsLoggedIn = loggedIn
	}
	return o.IsLoggedIn
}

func (o *GlobalOptions) checkOcBinary() bool {
	if !strings.Contains(o.OcBinary, string(os.PathSeparator)) {
		_, err := exec.LookPath(o.OcBinary)
		return err == nil
	}
	_, err := os.Stat(o.OcBinary)
	return !os.IsNotExist(err)
}

func (o *CompareOptions) check() error {
	if o.ContextDir != "." {
		if _, err := os.Stat(o.ContextDir); os.IsNotExist(err) {
			return fmt.Errorf("Context directory %s does not exist", o.ContextDir)
		}
	}
	// Check if template dir exists
	if o.TemplateDir != "." {
		td := o.ResolvedTemplateDir()
		if _, err := os.Stat(td); os.IsNotExist(err) {
			return fmt.Errorf("Template directory %s does not exist in context directory %s", td, o.ContextDir)
		}
	}
	// Check if param dir exists
	if o.ParamDir != "." {
		pd := o.ResolvedParamDir()
		if _, err := os.Stat(pd); os.IsNotExist(err) {
			return fmt.Errorf("Param directory %s does not exist in context directory %s", pd, o.ContextDir)
		}
	}

	if o.Format != "text" && o.Format != "json" {
		return errors.New("--diff must be either text or json")
	}
	if strings.Contains(o.Resource, "/") && len(o.Selector) > 0 {
		DebugMsg("Ignoring selector", o.Selector, "as resource is given")
		o.Selector = ""
	}

	return o.setNamespace()
}

// ResolvedTemplateDir returns template dir prefixed by the context dir.
func (o *CompareOptions) ResolvedTemplateDir() string {
	return utils.AbsoluteOrRelativePath(o.TemplateDir, o.ContextDir)
}

// ResolvedParamDir returns param dir prefixed by the context dir.
func (o *CompareOptions) ResolvedParamDir() string {
	return utils.AbsoluteOrRelativePath(o.ParamDir, o.ContextDir)
}

// ResolvedParamFiles returns param files prefixed by the context dir.
func (o *CompareOptions) ResolvedParamFiles() []string {
	files := []string{}
	for _, f := range o.ParamFiles {
		files = append(files, utils.AbsoluteOrRelativePath(f, o.ContextDir))
	}
	return files
}

// ResolvedPrivateKey returns private key prefixed by the context dir.
func (o *CompareOptions) ResolvedPrivateKey() string {
	return utils.AbsoluteOrRelativePath(o.PrivateKey, o.ContextDir)
}

func (o *CompareOptions) PathsToPreserve() []string {
	pathsToPreserve := []string{}
	if o.PreserveImmutableFields {
		pathsToPreserve = append(
			pathsToPreserve,
			"pvc:/spec/accessModes",
			"pvc:/spec/storageClassName",
			"pvc:/spec/resources/requests/storage",
			"route:/spec/host",
			"secret:/type",
		)
	}
	return append(pathsToPreserve, o.PreservePaths...)
}

func (o *ExportOptions) check() error {
	if strings.Contains(o.Resource, "/") && len(o.Selector) > 0 {
		DebugMsg("Ignoring selector", o.Selector, "as resource is given")
		o.Selector = ""
	}

	return o.setNamespace()
}

func (o *SecretsOptions) check() error {
	if len(o.ContextDirs) > 1 {
		return errors.New("secrets subcommand does not support multiple context directories")
	}
	return nil
}

// ResolvedParamDir returns param dir prefixed by the context dir.
func (o *SecretsOptions) ResolvedParamDir() string {
	return utils.AbsoluteOrRelativePath(o.ParamDir, o.ContextDir)
}

func (o *NamespaceOptions) setNamespace() error {
	if len(o.Namespace) == 0 {
		n, err := getOcNamespace()
		if err != nil {
			return err
		}
		o.Namespace = n
	} else {
		err := o.checkOcNamespace(o.Namespace)
		if err != nil {
			return fmt.Errorf("No such project: %s", o.Namespace)
		}
	}
	return nil
}

func (o *NamespaceOptions) checkOcNamespace(n string) error {
	if utils.Includes(o.CheckedNamespaces, n) {
		return nil
	}
	c := NewOcClient("")
	exists, err := c.CheckProjectExists(n)
	if exists {
		o.CheckedNamespaces = append(o.CheckedNamespaces, n)
	}
	return err
}

func getOcNamespace() (string, error) {
	c := NewOcClient("")
	return c.CurrentProject()
}

func getFileFlags(filename string, verbose bool) (map[string]string, error) {
	fileFlags := make(map[string]string)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if filename == "Tailorfile" {
			if verbose {
				PrintBluef("--> No file '%s' found.\n", filename)
			}
			return fileFlags, nil
		}
		return fileFlags, err
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return fileFlags, err
	}
	content := string(b)
	text := strings.TrimSuffix(content, "\n")
	lines := strings.Split(text, "\n")

	for _, untrimmedLine := range lines {
		line := strings.TrimSpace(untrimmedLine)
		if len(line) == 0 || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		pair := strings.SplitN(line, " ", 2)
		if len(pair) == 2 {
			key := pair[0]
			value := strings.TrimSpace(pair[1])
			if val, ok := fileFlags[key]; ok {
				value = val + "," + value
			}
			fileFlags[key] = value
		} else {
			fileFlags["resource"] = pair[0]
		}
	}
	return fileFlags, nil
}
