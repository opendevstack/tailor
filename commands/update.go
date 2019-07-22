package commands

import (
	"errors"
	"fmt"
	"io"

	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/openshift"
)

// Update prints the drift between desired and current state to STDOUT.
// If there is any, it asks for confirmation and applies the changeset.
func Update(compareOptions *cli.CompareOptions) error {
	updateRequired, changeset, err := calculateChangeset(compareOptions)
	if err != nil {
		return err
	}

	if updateRequired {
		if compareOptions.NonInteractive {
			err = apply(compareOptions, changeset)
			if err != nil {
				return fmt.Errorf("Update aborted: %s", err)
			}
		} else {
			c := cli.AskForConfirmation("Apply changes?")
			if c {
				fmt.Println("")
				err = apply(compareOptions, changeset)
				if err != nil {
					return fmt.Errorf("Update aborted: %s", err)
				}
			}
		}
	}

	return nil
}

func apply(compareOptions *cli.CompareOptions, c *openshift.Changeset) error {
	for _, change := range c.Create {
		err := ocCreate(change, compareOptions)
		if err != nil {
			return err
		}
	}

	for _, change := range c.Delete {
		err := ocDelete(change, compareOptions)
		if err != nil {
			return err
		}
	}

	for _, change := range c.Update {
		err := ocPatch(change, compareOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

func ocDelete(change *openshift.Change, compareOptions *cli.CompareOptions) error {
	kind := change.Kind
	name := change.Name
	fmt.Printf("Deleting %s/%s ... ", kind, name)
	args := []string{"delete", kind, name}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		"", // empty as name and selector is not allowed
	)
	_, errBytes, err := cli.RunCmd(cmd)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}
	return nil
}

func ocCreate(change *openshift.Change, compareOptions *cli.CompareOptions) error {
	kind := change.Kind
	name := change.Name
	config := change.DesiredState
	fmt.Printf("Creating %s/%s ... ", kind, name)
	args := []string{"create", "-f", "-"}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		compareOptions.Selector,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		defer stdin.Close()
		_, _ = io.WriteString(stdin, config)
	}()
	_, errBytes, err := cli.RunCmd(cmd)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}

	return nil
}

func ocPatch(change *openshift.Change, compareOptions *cli.CompareOptions) error {
	kind := change.Kind
	name := change.Name

	j := change.JsonPatches(false)

	fmt.Printf("Patching %s/%s ... ", kind, name)

	args := []string{"patch", kind + "/" + name, "--type=json", "--patch", j}
	cmd := cli.ExecOcCmd(
		args,
		compareOptions.Namespace,
		"", // empty as name and selector is not allowed
	)
	_, errBytes, err := cli.RunCmd(cmd)
	if err == nil {
		fmt.Println("done")
	} else {
		fmt.Println("failed")
		return errors.New(string(errBytes))
	}
	return nil
}
