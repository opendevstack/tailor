package openshift

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/utils"
	"golang.org/x/crypto/openpgp"
)

// DecryptedParams is used to edit/reveal secrets
func DecryptedParams(input, privateKey, passphrase string) (string, error) {
	c, err := newReadConverter(privateKey, passphrase)
	if err != nil {
		return "", err
	}
	return transformValues(input, []converterFunc{c.decrypt})
}

// EncodedParams is used to pass params to oc
func EncodedParams(input, privateKey, passphrase string) (string, error) {
	c, err := newReadConverter(privateKey, passphrase)
	if err != nil {
		return "", err
	}
	return transformValues(input, []converterFunc{c.decrypt, c.encode})
}

// EncryptedParams is used to save cleartext params to file
func EncryptedParams(input, previous, publicKeyDir, privateKey, passphrase string) (string, error) {
	c, err := newWriteConverter(previous, publicKeyDir, privateKey, passphrase)
	if err != nil {
		return "", err
	}
	return transformValues(input, []converterFunc{c.encrypt})
}

type paramConverter struct {
	PublicEntityList  openpgp.EntityList
	PrivateEntityList openpgp.EntityList
	PreviousParams    map[string]string
}

func (c *paramConverter) encode(key, val string) (string, string, error) {
	// If the value is already base64-encoded, we pass it through
	if strings.HasSuffix(key, ".B64") {
		return strings.TrimSuffix(key, ".B64"), val, nil
	}
	return key, base64.StdEncoding.EncodeToString([]byte(val)), nil
}

func (c *paramConverter) decode(key, val string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return key, "", err
	}
	return key, string(decoded), nil
}

// Decrypt given string
func (c *paramConverter) decrypt(key, val string) (string, string, error) {
	newVal, err := utils.Decrypt(val, c.PrivateEntityList)
	return key, newVal, err
}

// Encrypt encrypts given value. If the key was already present previously
// and the cleartext value did not change, then the previous encrypted string
// is returned.
func (c *paramConverter) encrypt(key, val string) (string, string, error) {
	if c.PreviousParams != nil {
		if _, exists := c.PreviousParams[key]; exists {
			previousEncryptedValue := c.PreviousParams[key]
			key, previousDecryptedValue, err := c.decrypt(key, previousEncryptedValue)
			if err != nil {
				// When decrypting fails, we display the error, but continue
				// as we can still encrypt ...
				cli.DebugMsg(err.Error())
			}
			if previousDecryptedValue == val {
				return key, previousEncryptedValue, nil
			}
		}
	}
	newVal, err := utils.Encrypt(val, c.PublicEntityList)
	return key, newVal, err
}

type converterFunc func(key, val string) (string, string, error)

func newReadConverter(privateKey, passphrase string) (*paramConverter, error) {
	el, err := utils.GetEntityList([]string{privateKey}, passphrase)
	if err != nil {
		return nil, err
	}
	return &paramConverter{PrivateEntityList: el}, nil
}

func newWriteConverter(previous, publicKeyDir, privateKey, passphrase string) (*paramConverter, error) {
	// Read previous params
	previousParams := map[string]string{}
	err := extractKeyValuePairs(previous, func(key, val string) error {
		previousParams[key] = val
		return nil
	}, func(line string) {})
	if err != nil {
		return nil, err
	}

	// Prefer "public-keys" folder over current directory
	if publicKeyDir == "." {
		if _, err := os.Stat("public-keys"); err == nil {
			publicKeyDir = "public-keys"
		}
	}

	// Read public keys
	cli.DebugMsg(fmt.Sprintf("Looking for public keys in '%s'", publicKeyDir))
	files, err := ioutil.ReadDir(publicKeyDir)
	if err != nil {
		return nil, err
	}
	filePattern := ".*\\.key$"
	keyFiles := []string{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "private.key") {
			continue
		}
		matched, _ := regexp.MatchString(filePattern, file.Name())
		if !matched {
			continue
		}
		keyFiles = append(keyFiles, publicKeyDir+string(os.PathSeparator)+file.Name())
	}
	if len(keyFiles) == 0 {
		return nil, fmt.Errorf(
			"No public key files found in '%s'. Files need to end in '.key'",
			publicKeyDir,
		)
	}

	publicEntityList, err := utils.GetEntityList(keyFiles, "")
	if err != nil {
		return nil, err
	}

	privateEntityList, err := utils.GetEntityList([]string{privateKey}, passphrase)
	if err != nil {
		return nil, err
	}

	return &paramConverter{
		PublicEntityList:  publicEntityList,
		PrivateEntityList: privateEntityList,
		PreviousParams:    previousParams,
	}, nil
}

func extractKeyValuePairs(input string, consumer func(key, val string) error, passthrough func(line string)) error {
	text := strings.TrimSuffix(input, "\n")
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			passthrough(line)
			continue
		}
		if strings.HasPrefix(line, "#") {
			cli.DebugMsg("Skipping comment:", line)
			passthrough(line)
			continue
		}
		pair := strings.SplitN(line, "=", 2)
		key := pair[0]
		val := ""
		if len(pair) > 1 {
			val = pair[1]
		}
		if err := consumer(key, val); err != nil {
			return err
		}
	}
	return nil
}

func transformValues(input string, converters []converterFunc) (string, error) {
	output := ""
	err := extractKeyValuePairs(input, func(key, val string) error {
		var err error
		for _, converter := range converters {
			key, val, err = converter(key, val)
			if err != nil {
				return err
			}
		}
		output = output + key + "=" + val + "\n"
		return nil
	}, func(line string) {
		output = output + line + "\n"
	})
	if err != nil {
		return "", err
	}
	return output, nil
}
