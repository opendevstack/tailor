package cli

import (
	"strings"
)

// openshiftVersion represents the client/server version pair.
type openshiftVersion struct {
	client string
	server string
}

// ExactMatch is true when client and server version are known and equal.
func (ov openshiftVersion) ExactMatch() bool {
	return !ov.Incomplete() && ov.client == ov.server
}

// Incomplete returns true if at least one version could not be detected properly.
func (ov openshiftVersion) Incomplete() bool {
	return ov.client == "?" || ov.server == "?"
}

// Get OC client and server version. See tests for example output of "oc version".
func ocVersion(ocClient OcClientVersioner) openshiftVersion {
	ov := openshiftVersion{"?", "?"}
	outBytes, errBytes, err := ocClient.Version()
	if err != nil {
		VerboseMsg("Failed to query client and server version, got:", string(errBytes))
		return ov
	}
	output := string(outBytes)

	ocClientVersion := ""
	ocServerVersion := ""
	extractVersion := func(versionPart string) string {
		ocVersionParts := strings.SplitN(versionPart, ".", 3)
		return strings.Join(ocVersionParts[:len(ocVersionParts)-1], ".")
	}

	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	for _, line := range lines {
		if len(line) > 0 {
			parts := strings.SplitN(line, " ", 2)
			if parts[0] == "oc" {
				ocClientVersion = extractVersion(parts[1])
			}
			if parts[0] == "openshift" {
				ocServerVersion = extractVersion(parts[1])
			}
		}
	}

	if len(ocClientVersion) > 0 && strings.Contains(ocClientVersion, ".") {
		ov.client = ocClientVersion
	}
	if len(ocServerVersion) > 0 && strings.Contains(ocServerVersion, ".") {
		ov.server = ocServerVersion
	}

	if ov.Incomplete() {
		VerboseMsg("Client and server version could not be detected properly, got:", output)
		return ov
	}
	return openshiftVersion{client: ocClientVersion, server: ocServerVersion}
}
