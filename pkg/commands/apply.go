package commands

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/openshift"
)

// Apply prints the drift between desired and current state to STDOUT.
// If there is any, it asks for confirmation and applies the changeset.
func Apply(nonInteractive bool, compareOptions *cli.CompareOptions) (bool, error) {
	ocClient := cli.NewOcClient(compareOptions.Namespace)
	var buf bytes.Buffer
	driftDetected, changeset, err := calculateChangeset(&buf, compareOptions, ocClient)
	fmt.Print(buf.String())
	if err != nil {
		return driftDetected, err
	}

	if driftDetected {
		if nonInteractive {
			err = apply(compareOptions, changeset)
			if err != nil {
				return driftDetected, fmt.Errorf("Apply aborted: %s", err)
			}
			if compareOptions.Verify {
				err := performVerification(compareOptions, ocClient)
				if err != nil {
					return true, err
				}
			}
			// As apply has run successfully, there should not be any drift
			// anymore. Therefore we report driftDetected=false here.
			return false, nil
		}

		c := cli.AskForConfirmation("Apply changes?")
		if c {
			fmt.Println("")
			err = apply(compareOptions, changeset)
			if err != nil {
				return driftDetected, fmt.Errorf("Apply aborted: %s", err)
			}
			if compareOptions.Verify {
				err := performVerification(compareOptions, ocClient)
				if err != nil {
					return true, err
				}
			}
			// As apply has run successfully, there should not be any drift
			// anymore. Therefore we report driftDetected=false here.
			return false, nil
		}
		// Changes were not applied, so we report if drift was detected.
		return driftDetected, nil
	}

	// No drift, nothing to do ...
	return false, nil
}

func apply(compareOptions *cli.CompareOptions, c *openshift.Changeset) error {
	ocClient := cli.NewOcClient(compareOptions.Namespace)

	for _, change := range c.Create {
		err := ocApply("Creating", change, compareOptions, ocClient)
		if err != nil {
			return err
		}
	}

	for _, change := range c.Delete {
		err := ocDelete(change, compareOptions, ocClient)
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

func ocDelete(change *openshift.Change, compareOptions *cli.CompareOptions, ocClient cli.OcClientDeleter) error {
	fmt.Printf("Deleting %s ... ", change.ItemName())
	errBytes, err := ocClient.Delete(change.Kind, change.Name)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}
	return nil
}

func ocApply(label string, change *openshift.Change, compareOptions *cli.CompareOptions, ocClient cli.OcClientApplier) error {
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
