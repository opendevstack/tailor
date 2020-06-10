package cli

import (
	"bufio"
	"bytes"
	"testing"
)

func TestAskForAction(t *testing.T) {
	tests := map[string]struct {
		input          string
		options        []string
		expectedAnswer string
	}{
		"input of key of one option": {
			input:          "y\n",
			options:        []string{"y=yes", "n=no"},
			expectedAnswer: "y",
		},
		"input of value of one option": {
			input:          "yes\n",
			options:        []string{"y=yes", "n=no"},
			expectedAnswer: "y",
		},
		"input of key with uppercase character": {
			input:          "Y\n",
			options:        []string{"y=yes", "n=no"},
			expectedAnswer: "y",
		},
		"input of value with uppercase character": {
			input:          "Yes\n",
			options:        []string{"y=yes", "n=no"},
			expectedAnswer: "y",
		},
		"input of invalid answer at first": {
			input:          "m\nn\n",
			options:        []string{"y=yes", "n=no"},
			expectedAnswer: "n",
		},
		"empty input at first": {
			input:          "\ny\n",
			options:        []string{"y=yes", "n=no"},
			expectedAnswer: "y",
		},
		"input with surrounding space": {
			input:          " no \n",
			options:        []string{"y=yes", "n=no"},
			expectedAnswer: "n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var stdin bytes.Buffer
			stdin.Write([]byte(tc.input))
			stdinReader := bufio.NewReader(&stdin)
			a := AskForAction("What?", tc.options, stdinReader)
			if a != tc.expectedAnswer {
				t.Fatalf("Want: '%s', got: '%s'", tc.expectedAnswer, a)
			}
		})
	}
}
