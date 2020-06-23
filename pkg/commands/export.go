package commands

import (
	"fmt"

	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/openshift"
)

// Export prints an export of targeted resources to STDOUT.
func Export(exportOptions *cli.ExportOptions) error {
	filter, err := openshift.NewResourceFilter(exportOptions.Resource, exportOptions.Selector, exportOptions.Excludes)
	if err != nil {
		return err
	}

	c := cli.NewOcClient(exportOptions.Namespace)
	out, err := openshift.ExportAsTemplateFile(
		filter,
		exportOptions.WithAnnotations,
		exportOptions.Namespace,
		exportOptions.WithHardcodedNamespace,
		exportOptions.TrimAnnotations,
		c,
	)
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
