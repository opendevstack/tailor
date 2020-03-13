package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/openshift"
)

// Diff prints the drift between desired and current state to STDOUT.
func Diff(compareOptions *cli.CompareOptions) (bool, error) {
	ocClient := cli.NewOcClient(compareOptions.Namespace)
	var buf bytes.Buffer
	driftDetected, _, err := calculateChangeset(&buf, compareOptions, ocClient)
	fmt.Print(buf.String())
	return driftDetected, err
}

func calculateChangeset(w io.Writer, compareOptions *cli.CompareOptions, ocClient cli.ClientProcessorExporter) (bool, *openshift.Changeset, error) {
	updateRequired := false

	where := compareOptions.TemplateDir

	fmt.Fprintf(w,
		"Comparing templates in %s with OCP namespace %s.\n",
		where,
		compareOptions.Namespace,
	)

	if len(compareOptions.Resource) > 0 && len(compareOptions.Selector) > 0 {
		fmt.Fprintf(w,
			"Limiting resources to %s with selector %s.\n",
			compareOptions.Resource,
			compareOptions.Selector,
		)
	} else if len(compareOptions.Selector) > 0 {
		fmt.Fprintf(w,
			"Limiting to resources with selector %s.\n",
			compareOptions.Selector,
		)
	} else if len(compareOptions.Resource) > 0 {
		fmt.Fprintf(w,
			"Limiting resources to %s.\n",
			compareOptions.Resource,
		)
	}

	resource := compareOptions.Resource

	filter, err := openshift.NewResourceFilter(resource, compareOptions.Selector, compareOptions.Exclude)
	if err != nil {
		return updateRequired, &openshift.Changeset{}, err
	}

	templateBasedList, err := assembleTemplateBasedResourceList(
		filter,
		compareOptions,
		ocClient,
	)
	if err != nil {
		return updateRequired, &openshift.Changeset{}, err
	}

	platformBasedList, err := assemblePlatformBasedResourceList(filter, compareOptions, ocClient)
	if err != nil {
		return updateRequired, &openshift.Changeset{}, err
	}

	platformResourcesWord := "resources"
	if platformBasedList.Length() == 1 {
		platformResourcesWord = "resource"
	}
	templateResourcesWord := "resources"
	if templateBasedList.Length() == 1 {
		templateResourcesWord = "resource"
	}
	fmt.Fprintf(w,
		"Found %d %s in OCP cluster (current state) and %d %s in processed templates (desired state).\n\n",
		platformBasedList.Length(),
		platformResourcesWord,
		templateBasedList.Length(),
		templateResourcesWord,
	)

	if templateBasedList.Length() == 0 && !compareOptions.Force {
		fmt.Fprint(w, "No items where found in desired state. ")
		if len(compareOptions.Resource) == 0 && len(compareOptions.Selector) == 0 {
			fmt.Fprintf(w,
				"Are there any templates in %s?\n",
				where,
			)
		} else {
			fmt.Fprintf(w,
				"Possible reasons are:\n"+
					"* No templates are located in %s\n",
				where,
			)
			if len(compareOptions.Resource) > 0 {
				fmt.Fprintf(w,
					"* No templates contain resources of kinds: %s\n",
					compareOptions.Resource,
				)
			}
			if len(compareOptions.Selector) > 0 {
				fmt.Fprintf(w,
					"* No templates contain resources matching selector: %s\n",
					compareOptions.Selector,
				)
			}
		}
		fmt.Fprintln(w, "\nRefusing to continue without --force")
		return updateRequired, &openshift.Changeset{}, errors.New("Diff not performed due to misconfiguration")
	}

	changeset, err := compare(
		w,
		platformBasedList,
		templateBasedList,
		compareOptions.UpsertOnly,
		compareOptions.AllowRecreate,
		compareOptions.RevealSecrets,
		compareOptions.PathsToPreserve(),
	)
	if err != nil {
		return false, changeset, err
	}
	updateRequired = !changeset.Blank()
	return updateRequired, changeset, nil
}

func compare(w io.Writer, remoteResourceList *openshift.ResourceList, localResourceList *openshift.ResourceList, upsertOnly bool, allowRecreate bool, revealSecrets bool, preservePaths []string) (*openshift.Changeset, error) {
	changeset, err := openshift.NewChangeset(remoteResourceList, localResourceList, upsertOnly, allowRecreate, preservePaths)
	if err != nil {
		return changeset, err
	}

	for _, change := range changeset.Noop {
		fmt.Fprintf(w, "* %s is in sync\n", change.ItemName())
	}

	for _, change := range changeset.Delete {
		cli.FprintRedf(w, "- %s to delete\n", change.ItemName())
		fmt.Fprint(w, change.Diff(revealSecrets))
	}

	for _, change := range changeset.Create {
		cli.FprintGreenf(w, "+ %s to create\n", change.ItemName())
		fmt.Fprint(w, change.Diff(revealSecrets))
	}

	for _, change := range changeset.Update {
		cli.FprintYellowf(w, "~ %s to update\n", change.ItemName())
		fmt.Fprint(w, change.Diff(revealSecrets))
	}

	fmt.Fprintf(w, "\nSummary: %d in sync, ", len(changeset.Noop))
	cli.FprintGreenf(w, "%d to create", len(changeset.Create))
	fmt.Fprint(w, ", ")
	cli.FprintYellowf(w, "%d to update", len(changeset.Update))
	fmt.Fprint(w, ", ")
	cli.FprintRedf(w, "%d to delete\n\n", len(changeset.Delete))

	return changeset, nil
}

func assembleTemplateBasedResourceList(filter *openshift.ResourceFilter, compareOptions *cli.CompareOptions, ocClient cli.OcClientProcessor) (*openshift.ResourceList, error) {
	var inputs [][]byte

	files, err := ioutil.ReadDir(compareOptions.TemplateDir)
	if err != nil {
		return nil, err
	}
	filePattern := ".*\\.ya?ml$"
	re := regexp.MustCompile(filePattern)
	for _, file := range files {
		matched := re.MatchString(file.Name())
		if !matched {
			continue
		}
		cli.DebugMsg("Reading template", file.Name())
		processedOut, err := openshift.ProcessTemplate(
			compareOptions.TemplateDir,
			file.Name(),
			compareOptions.ParamDir,
			compareOptions,
			ocClient,
		)
		if err != nil {
			return nil, fmt.Errorf("Could not process %s template: %s", file.Name(), err)
		}
		inputs = append(inputs, processedOut)
	}

	return openshift.NewTemplateBasedResourceList(filter, inputs...)
}

func assemblePlatformBasedResourceList(filter *openshift.ResourceFilter, compareOptions *cli.CompareOptions, ocClient cli.OcClientExporter) (*openshift.ResourceList, error) {
	exportedOut, err := ocClient.Export(filter.ConvertToKinds(), filter.Label)
	if err != nil {
		return nil, fmt.Errorf("Could not export %s resources: %s", filter.String(), err)
	}
	return openshift.NewPlatformBasedResourceList(filter, exportedOut)
}
