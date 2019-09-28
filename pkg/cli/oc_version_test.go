package cli

import (
	"testing"

	"github.com/opendevstack/tailor/internal/test/helper"
)

type mockOcVersionClient struct {
	t       *testing.T
	fixture string
}

func (c *mockOcVersionClient) Version() ([]byte, []byte, error) {
	content := helper.ReadFixtureFile(c.t, "version/"+c.fixture)
	return content, []byte(""), nil
}

func TestOcVersion(t *testing.T) {
	tests := map[string]struct {
		fixture        string
		expectedClient string
		expectedServer string
	}{
		"client=3.9 and server=3.11": {
			fixture:        "client-3_9-and-server-3_11.txt",
			expectedClient: "v3.9",
			expectedServer: "v3.11",
		},
		"client=3.11 and server=3.11": {
			fixture:        "client-3_11-and-server-3_11.txt",
			expectedClient: "v3.11",
			expectedServer: "v3.11",
		},
		"client=3.11 and server=?": {
			fixture:        "client-3_11-and-server-unknown.txt",
			expectedClient: "v3.9",
			expectedServer: "?",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &mockOcVersionClient{t: t, fixture: tc.fixture}
			ov := ocVersion(c)
			if ov.client != tc.expectedClient {
				t.Fatalf("Expected client version: '%s', got: '%s'", tc.expectedClient, ov.client)
			}
			if ov.server != tc.expectedServer {
				t.Fatalf("Expected client version: '%s', got: '%s'", tc.expectedServer, ov.server)
			}
		})
	}
}
