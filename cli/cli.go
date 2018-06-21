package cli

import (
	"bufio"
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

var options *Options

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
	options = &Options{
		Verbose:   false,
		Namespace: "",
		Selector:  "",
	}
}

func SetOptions(verbose bool, namespace string, selector string) {
	options = &Options{
		Verbose:   verbose,
		Namespace: namespace,
		Selector:  selector,
	}
}

func VerboseMsg(messages ...string) {
	if options.Verbose {
		PrintBluef("--> %s\n", strings.Join(messages, " "))
	}
}

func ExecOcCmd(args []string) *exec.Cmd {
	if len(options.Namespace) > 0 {
		args = append(args, "--namespace="+options.Namespace)
	}
	if len(options.Selector) > 0 {
		args = append(args, "--selector="+options.Selector)
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

func ShowDiff(versions []string) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(versions[0]),
		B:        difflib.SplitLines(versions[1]),
		FromFile: "Remote State",
		ToFile:   "Local Config",
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
	cmd := exec.Command(editor, ".ENV.DEC")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
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
