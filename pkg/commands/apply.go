package commands

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/openshift"
)

type printChange func(w io.Writer, change *openshift.Change, revealSecrets bool)
type handleChange func(label string, change *openshift.Change, compareOptions *cli.CompareOptions, ocClient cli.ClientModifier) error

// Apply prints the drift between desired and current state to STDOUT.
// If there is any, it asks for confirmation and applies the changeset.
func Apply(nonInteractive bool, compareOptions *cli.CompareOptions, ocClient cli.ClientApplier, stdin io.Reader) (bool, error) {
	stdinReader := bufio.NewReader(stdin)

	var buf bytes.Buffer
	driftDetected, changeset, err := calculateChangeset(&buf, compareOptions, ocClient)
	fmt.Print(buf.String())
	if err != nil {
		return driftDetected, err
	}

	if driftDetected {
		if nonInteractive {
			err = apply(compareOptions, changeset, ocClient)
			if err != nil {
				return true, fmt.Errorf("Apply aborted: %s", err)
			}
			if compareOptions.Verify {
				err := performVerification(compareOptions, ocClient)
				if err != nil {
					return true, err
				}
			}
			// As apply has run successfully, there should not be any drift
			// anymore. Therefore we report no drift here.
			return false, nil
		}

		options := []string{"y=yes", "n=no"}
		// Selecting makes no sense when --verify is given, as the verification
		// would fail if not all changes are selected.
		// Selecting is also pointless if there is only one change in total.
		allowSelecting := !compareOptions.Verify && !changeset.ExactlyOne()
		if allowSelecting {
			options = append(options, "s=select")
		}
		a := cli.AskForAction("Apply all changes?", options, stdinReader)
		if a == "y" {
			fmt.Println("")
			err = apply(compareOptions, changeset, ocClient)
			if err != nil {
				return true, fmt.Errorf("Apply aborted: %s", err)
			}
			if compareOptions.Verify {
				err := performVerification(compareOptions, ocClient)
				if err != nil {
					return true, err
				}
			}
			// As apply has run successfully, there should not be any drift
			// anymore. Therefore we report no drift here.
			return false, nil
		} else if allowSelecting && a == "s" {
			anyChangeSkipped := false

			anyDeleteChangeSkipped, err := askAndApply(compareOptions, ocClient, stdinReader, changeset.Delete, printDeleteChange, "Deleting", ocDelete)
			if err != nil {
				return true, fmt.Errorf("Apply aborted: %s", err)
			} else if anyDeleteChangeSkipped {
				anyChangeSkipped = true
			}
			anyCreateChangeSkipped, err := askAndApply(compareOptions, ocClient, stdinReader, changeset.Create, printCreateChange, "Creating", ocApply)
			if err != nil {
				return true, fmt.Errorf("Apply aborted: %s", err)
			} else if anyCreateChangeSkipped {
				anyChangeSkipped = true
			}
			anyUpdateChangeSkipped, err := askAndApply(compareOptions, ocClient, stdinReader, changeset.Update, printUpdateChange, "Updating", ocApply)
			if err != nil {
				return true, fmt.Errorf("Apply aborted: %s", err)
			} else if anyUpdateChangeSkipped {
				anyChangeSkipped = true
			}

			return anyChangeSkipped, nil
		}

		// Changes were not applied, so we report that drift was detected.
		return true, nil
	}

	// No drift, nothing to do ...
	return false, nil
}

func askAndApply(compareOptions *cli.CompareOptions, ocClient cli.ClientApplier, stdinReader *bufio.Reader, changes []*openshift.Change, changePrinter printChange, label string, changeHandler handleChange) (bool, error) {
	anyChangeSkipped := false

	for _, change := range changes {
		fmt.Println("")
		var buf bytes.Buffer
		changePrinter(&buf, change, compareOptions.RevealSecrets)
		fmt.Print(buf.String())
		a := cli.AskForAction(
			fmt.Sprintf("Apply change to %s?", change.ItemName()),
			[]string{"y=yes", "n=no"},
			stdinReader,
		)
		if a == "y" {
			fmt.Println("")
			err := changeHandler(label, change, compareOptions, ocClient)
			if err != nil {
				return true, fmt.Errorf("Apply aborted: %s", err)
			}
		} else {
			anyChangeSkipped = true
		}
	}
	return anyChangeSkipped, nil
}

func apply(compareOptions *cli.CompareOptions, c *openshift.Changeset, ocClient cli.ClientModifier) error {

	for _, change := range c.Create {
		err := ocApply("Creating", change, compareOptions, ocClient)
		if err != nil {
			return err
		}
	}

	for _, change := range c.Delete {
		err := ocDelete("Deleting", change, compareOptions, ocClient)
		if err != nil {
			return err
		}
	}

	for _, change := range c.Update {
		err := ocApply("Updating", change, compareOptions, ocClient)
		if err != nil {
			return err
		}
	}

	return nil
}

func ocDelete(label string, change *openshift.Change, compareOptions *cli.CompareOptions, ocClient cli.ClientModifier) error {
	fmt.Printf("%s %s ... ", label, change.ItemName())
	errBytes, err := ocClient.Delete(change.Kind, change.Name)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}
	return nil
}

func ocApply(label string, change *openshift.Change, compareOptions *cli.CompareOptions, ocClient cli.ClientModifier) error {
	fmt.Printf("%s %s ... ", label, change.ItemName())
	errBytes, err := ocClient.Apply(change.DesiredState, compareOptions.Selector)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}

	return nil
}

func performVerification(compareOptions *cli.CompareOptions, ocClient cli.ClientProcessorExporter) error {
	var buf bytes.Buffer
	fmt.Print("\nVerifying current state matches desired state ... ")
	driftDetected, _, err := calculateChangeset(&buf, compareOptions, ocClient)
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}
	if driftDetected {
		fmt.Print("failed! Detected drift:\n\n")
		fmt.Println(buf.String())
		return errors.New("Verification failed")
	}
	fmt.Println("successful")
	return nil
}
