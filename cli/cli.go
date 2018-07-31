package cli

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/pmezard/go-difflib/difflib"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Options struct {
	Verbose   bool
	Namespace string
	Selector  string
}

var verbose bool
var debug bool

var PrintGreenf func(format string, a ...interface{})
var PrintBluef func(format string, a ...interface{})
var PrintYellowf func(format string, a ...interface{})
var PrintRedf func(format string, a ...interface{})

func init() {
	color.Output = os.Stderr
	PrintGreenf = color.New(color.FgGreen).PrintfFunc()
	PrintBluef = color.New(color.FgBlue).PrintfFunc()
	PrintYellowf = color.New(color.FgYellow).PrintfFunc()
	PrintRedf = color.New(color.FgRed).PrintfFunc()
	verbose = false
}

func GetOcNamespace() (string, error) {
	cmd := ExecPlainOcCmd([]string{"project", "--short"})
	n, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(n)), err
}

func VerboseMsg(messages ...string) {
	if verbose {
		PrintBluef("--> %s\n", strings.Join(messages, " "))
	}
}

func DebugMsg(messages ...string) {
	if debug {
		PrintBluef("--> %s\n", strings.Join(messages, " "))
	}
}

func ExecOcCmd(args []string, namespace string, selector string) *exec.Cmd {
	if len(namespace) > 0 {
		args = append(args, "--namespace="+namespace)
	}
	if len(selector) > 0 {
		args = append(args, "--selector="+selector)
	}
	return ExecPlainOcCmd(args)
}

func ExecPlainOcCmd(args []string) *exec.Cmd {
	return execCmd("oc", args)
}

func execCmd(executable string, args []string) *exec.Cmd {
	VerboseMsg(executable + " " + strings.Join(args, " "))
	return exec.Command(executable, args...)
}

func ShowDiff(a string, b string) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(a),
		B:        difflib.SplitLines(b),
		FromFile: "Current State (OpenShift cluster)",
		ToFile:   "Desired State (Processed template)",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	fmt.Printf(text)
}

// askForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
func AskForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func EditEnvFile(content string) (string, error) {
	ioutil.WriteFile(".ENV.DEC", []byte(content), 0644)
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	_, err := exec.LookPath(editor)
	if err != nil {
		return "", errors.New(
			fmt.Sprintf("Please install '%s' or set/change $EDITOR", editor),
		)
	}

	cmd := exec.Command(editor, ".ENV.DEC")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadFile(".ENV.DEC")
	if err != nil {
		return "", err
	}
	os.Remove(".ENV.DEC")
	return string(data), nil
}
