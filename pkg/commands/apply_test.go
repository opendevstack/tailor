package commands

import (
	"bytes"
	"testing"

	"github.com/opendevstack/tailor/internal/test/helper"
	"github.com/opendevstack/tailor/pkg/cli"
	"github.com/opendevstack/tailor/pkg/utils"
)

type mockOcApplyClient struct {
	t              *testing.T
	currentFixture string
	desiredFixture string
}

func (c *mockOcApplyClient) Export(target string, label string) ([]byte, error) {
	return helper.ReadFixtureFile(c.t, "command-apply/"+c.currentFixture), nil
}

func (c *mockOcApplyClient) Process(args []string) ([]byte, []byte, error) {
	return helper.ReadFixtureFile(c.t, "command-apply/"+c.desiredFixture), []byte(""), nil
}

func (c *mockOcApplyClient) Apply(config string, selector string) ([]byte, error) {
	return []byte(""), nil
}

func (c *mockOcApplyClient) Delete(kind string, name string) ([]byte, error) {
	return []byte(""), nil
}

func TestApply(t *testing.T) {
	tests := map[string]struct {
		namespace      string
		nonInteractive bool
		stdinInput     string
		currentFixture string
		desiredFixture string
		expectedDrift  bool
	}{
		"non-interactively": {
			namespace:      "foo",
			nonInteractive: true,
			stdinInput:     "",
			currentFixture: "current-list.yml",
			desiredFixture: "template-dir/desired-list.yml",
			expectedDrift:  false,
		},
		"interactively": {
			namespace:      "foo",
			nonInteractive: false,
			stdinInput:     "y\n",
			currentFixture: "current-list.yml",
			desiredFixture: "template-dir/desired-list.yml",
			expectedDrift:  false,
		},
		"interactively with select": {
			namespace:      "foo",
			nonInteractive: false,
			stdinInput:     "s\ny\nn\n",
			currentFixture: "current-list.yml",
			desiredFixture: "template-dir/desired-list.yml",
			expectedDrift:  true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			globalOptions := cli.InitGlobalOptions(&utils.OsFS{})
			compareOptions := &cli.CompareOptions{
				GlobalOptions:    globalOptions,
				NamespaceOptions: &cli.NamespaceOptions{Namespace: tc.namespace},
				TemplateDir:      "../../internal/test/fixtures/command-apply/template-dir",
				ParamFiles:       []string{},
			}
			ocClient := &mockOcApplyClient{
				currentFixture: tc.currentFixture,
				desiredFixture: tc.desiredFixture,
			}
			var stdin bytes.Buffer
			stdin.Write([]byte(tc.stdinInput))
			drift, err := Apply(tc.nonInteractive, compareOptions, ocClient, &stdin)
			if err != nil {
				t.Fatal(err)
			}
			if drift != tc.expectedDrift {
				t.Fatalf("Want drift=%t, got drift=%t\n", tc.expectedDrift, drift)
			}
		})
	}
}
