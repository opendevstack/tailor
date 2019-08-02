package openshift

import (
	"testing"
)

func TestTemplateContainsTailorNamespaceParam(t *testing.T) {
	contains, err := templateContainsTailorNamespaceParam("../../internal/test/fixtures/template-with-tailor-namespace-param.yml")
	if err != nil {
		t.Errorf("Could not determine if the template contains the param: %s", err)
	}
	if !contains {
		t.Error("Template contains param, but it was not detected")
	}
}
