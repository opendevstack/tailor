package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
)

var verbose bool
var debug bool
var ocBinary string

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
	return execCmd(ocBinary, args)
}

// RunCmd runs the given command and returns the result
func RunCmd(cmd *exec.Cmd) (outBytes, errBytes []byte, err error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	outBytes = stdout.Bytes()
	errBytes = stderr.Bytes()
	return outBytes, errBytes, err
}

func execCmd(executable string, args []string) *exec.Cmd {
	VerboseMsg(executable + " " + strings.Join(args, " "))
	return exec.Command(executable, args...)
}

// askForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
func AskForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

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
	err := ioutil.WriteFile(".ENV.DEC", []byte(content), 0644)
	if err != nil {
		return "", err
	}
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	_, err = exec.LookPath(editor)
	if err != nil {
		return "", fmt.Errorf(
			"Please install '%s' or set/change $EDITOR",
			editor,
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
