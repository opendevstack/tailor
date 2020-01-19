package commands

import (
	"errors"
	"fmt"

	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/openshift"
)

// Apply prints the drift between desired and current state to STDOUT.
// If there is any, it asks for confirmation and applies the changeset.
func Apply(nonInteractive bool, compareOptionSets map[string]*cli.CompareOptions) error {
	updateRequired, changesets, err := calculateChangesets(compareOptionSets)
	if err != nil {
		return err
	}

	if updateRequired {
		if nonInteractive {
			for contextDir, compareOptions := range compareOptionSets {
				err = apply(compareOptions, changesets[contextDir])
				if err != nil {
					return fmt.Errorf("Apply aborted: %s", err)
				}
			}
		} else {
			c := cli.AskForConfirmation("Apply changes?")
			if c {
				fmt.Println("")
				for contextDir, compareOptions := range compareOptionSets {
					err = apply(compareOptions, changesets[contextDir])
					if err != nil {
						return fmt.Errorf("Apply aborted: %s", err)
					}
				}
			}
		}
	}

	return nil
}

func apply(compareOptions *cli.CompareOptions, c *openshift.Changeset) error {
	ocClient := cli.NewOcClient(compareOptions.Namespace)
	fmt.Printf(
		"===== Applying changes related to context directory %s =====\n",
		compareOptions.ContextDir,
	)

	for _, change := range c.Create {
		err := ocCreate(change, compareOptions, ocClient)
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
		err := ocPatch(change, compareOptions, ocClient)
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

func ocCreate(change *openshift.Change, compareOptions *cli.CompareOptions, ocClient cli.OcClientCreator) error {
	fmt.Printf("Creating %s ... ", change.ItemName())
	errBytes, err := ocClient.Create(change.DesiredState, compareOptions.Selector)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}

	return nil
}

func ocPatch(change *openshift.Change, compareOptions *cli.CompareOptions, ocClient cli.OcClientPatcher) error {
	fmt.Printf("Patching %s ... ", change.ItemName())
	errBytes, err := ocClient.Patch(change.ItemName(), change.JSONPatches())
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}
	return nil
}
