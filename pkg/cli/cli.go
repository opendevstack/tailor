package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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

// PrintGreenf prints in green.
var PrintGreenf func(format string, a ...interface{})

// FprintGreenf prints in green to w.
var FprintGreenf func(w io.Writer, format string, a ...interface{})

// PrintBluef prints in blue.
var PrintBluef func(format string, a ...interface{})

// FprintBluef prints in green to w.
var FprintBluef func(w io.Writer, format string, a ...interface{})

// PrintYellowf prints in yellow.
var PrintYellowf func(format string, a ...interface{})

// FprintYellowf prints in green to w.
var FprintYellowf func(w io.Writer, format string, a ...interface{})

// PrintRedf prints in red.
var PrintRedf func(format string, a ...interface{})

// FprintRedf prints in green to w.
var FprintRedf func(w io.Writer, format string, a ...interface{})

func init() {
	color.Output = os.Stderr
	PrintGreenf = color.New(color.FgGreen).PrintfFunc()
	PrintBluef = color.New(color.FgBlue).PrintfFunc()
	PrintYellowf = color.New(color.FgYellow).PrintfFunc()
	PrintRedf = color.New(color.FgRed).PrintfFunc()
	FprintGreenf = color.New(color.FgGreen).FprintfFunc()
	FprintBluef = color.New(color.FgBlue).FprintfFunc()
	FprintYellowf = color.New(color.FgYellow).FprintfFunc()
	FprintRedf = color.New(color.FgRed).FprintfFunc()
	verbose = false
}

// VerboseMsg prints given message when verbose mode is on.
// Verbose mode is implicitly turned on when debug mode is on.
func VerboseMsg(messages ...string) {
	if verbose {
		PrintBluef("--> %s\n", strings.Join(messages, " "))
	}
}

// DebugMsg prints given message when debug mode is on.
func DebugMsg(messages ...string) {
	if debug {
		PrintBluef("--> %s\n", strings.Join(messages, " "))
	}
}

// ExecOcCmd executes "oc" with given namespace and selector applied.
func ExecOcCmd(args []string, namespace string, selector string) *exec.Cmd {
	if len(namespace) > 0 {
		args = append(args, "--namespace="+namespace)
	}
	if len(selector) > 0 {
		args = append(args, "--selector="+selector)
	}
	return ExecPlainOcCmd(args)
}

// ExecPlainOcCmd executes "oc" with given arguments applied.
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

// AskForAction asks the user the given question. A user must type in one of the presented options and
// then press enter.If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
// Options are of form "y=yes". The matching is fuzzy, which means allowed values are
// "y", "Y", "yes", "YES", "Yes" and so on. The returned value is always the "key" ("y" in this case),
// regardless if the input was "y" or "yes" etc.
func AskForAction(question string, options []string, reader *bufio.Reader) string {
	validAnswers := map[string]string{}
	for _, v := range options {
		p := strings.Split(v, "=")
		validAnswers[p[0]] = p[0]
		validAnswers[p[1]] = p[0]
	}

	for {
		fmt.Printf("%s [%s]: ", question, strings.Join(options, ", "))

		answer, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		answer = strings.ToLower(strings.TrimSpace(answer))

		if v, ok := validAnswers[answer]; !ok {
			fmt.Printf("'%s' is not a valid option. Please try again.\n", answer)
		} else {
			return v
		}
	}
}

// EditEnvFile opens content in EDITOR, and returns saved content.
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
