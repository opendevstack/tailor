package commands

import (
	"errors"
	"fmt"

	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/openshift"
)

// Apply prints the drift between desired and current state to STDOUT.
// If there is any, it asks for confirmation and applies the changeset.
func Apply(nonInteractive bool, compareOptionSets map[string]*cli.CompareOptions) (bool, error) {
	driftDetected, changesets, err := calculateChangesets(compareOptionSets)
	if err != nil {
		return driftDetected, err
	}

	if driftDetected {
		if nonInteractive {
			for contextDir, compareOptions := range compareOptionSets {
				err = apply(compareOptions, changesets[contextDir])
				if err != nil {
					return driftDetected, fmt.Errorf("Apply aborted: %s", err)
				}
			}
			// As apply has run successfully, there should not be any drift
			// anymore. Therefore we report driftDetected=false here.
			return false, nil
		}

		c := cli.AskForConfirmation("Apply changes?")
		if c {
			fmt.Println("")
			for contextDir, compareOptions := range compareOptionSets {
				err = apply(compareOptions, changesets[contextDir])
				if err != nil {
					return driftDetected, fmt.Errorf("Apply aborted: %s", err)
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
	fmt.Printf(
		"===== Applying changes related to context directory %s =====\n",
		compareOptions.ContextDir,
	)

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
