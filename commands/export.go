package commands

import (
	"fmt"

	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/openshift"
)

// Export prints an export of targeted resources to STDOUT.
func Export(exportOptions *cli.ExportOptions) error {
	filter, err := openshift.NewResourceFilter(exportOptions.Resource, exportOptions.Selector, exportOptions.Exclude)
	if err != nil {
		return err
	}

	out, err := openshift.ExportAsTemplateFile(filter, exportOptions)
	if err != nil {
		return fmt.Errorf(
			"Could not export %s resources as template: %s",
			filter.String(),
			err,
		)
	}

	fmt.Println(out)
	return nil
}
