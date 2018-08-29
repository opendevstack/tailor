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

type Param struct {
	Key       string
	Value     string
	IsSecret  bool
	Decrypted string
}

type Params []*Param

func NewParams(content, privateKey, passphrase string) (Params, error) {
	params := Params{}
	entityList := openpgp.EntityList{}
	text := strings.TrimSuffix(content, "\n")
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		pair := strings.SplitN(line, "=", 2)
		key := pair[0]
		value := pair[1]
		param := &Param{}

		// If the key ends with .STRING, base64 encode the value and then
		// change the key to end with .ENC to trigger the next step.
		if strings.HasSuffix(key, ".STRING") {
			key = strings.Replace(key, ".STRING", ".ENC", -1)
			cli.DebugMsg("Encountered STRING param", key)
			value = base64.StdEncoding.EncodeToString([]byte(value))
		}

		if strings.HasSuffix(key, ".ENC") {
			param.IsSecret = true
			param.Key = strings.Replace(key, ".ENC", "", -1)
			cli.DebugMsg("Encountered ENC param", param.Key)
			if len(privateKey) > 0 {
				if len(entityList) == 0 {
					el, err := utils.GetEntityList([]string{privateKey}, passphrase)
					if err != nil {
						return nil, err
					}
					entityList = el
				}
				param.Value = value
				decrypted, err := utils.Decrypt(value, entityList)
				if err != nil {
					return params, err
				}
				param.Decrypted = decrypted
			} else {
				param.Decrypted = value
			}
		} else {
			cli.DebugMsg("Encountered RAW param", key)
			param.IsSecret = false
			param.Value = value
			param.Key = key
		}

		params = append(params, param)
	}

	return params, nil
}

func NewParamsFromInput(content string) (Params, error) {
	cli.DebugMsg("Reading params from input")
	return NewParams(content, "", "")
}

func NewParamsFromFile(filename, privateKey, passphrase string) (Params, error) {
	cli.DebugMsg("Reading params from file", filename)
	content := ""
	if _, err := os.Stat(filename); err == nil {
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		content = string(bytes)
	}
	return NewParams(content, privateKey, passphrase)
}

func (p Params) String() string {
	out := ""
	for _, param := range p {
		var val string
		if param.IsSecret {
			val = param.Decrypted
		} else {
			val = param.Value
		}
		out = out + param.Key + "=" + val + "\n"
	}
	return out
}

// Encrypt params and create string from them
func (p Params) Render(publicKeyDir string, previousParams Params) (string, error) {
	out := ""

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
		return "", err
	}
	filePattern := ".*\\.key$"
	keyFiles := []string{}
	for _, file := range files {
		if file.Name() == "private.key" {
			continue
		}
		matched, _ := regexp.MatchString(filePattern, file.Name())
		if !matched {
			continue
		}
		keyFiles = append(keyFiles, publicKeyDir+string(os.PathSeparator)+file.Name())
	}
	if len(keyFiles) == 0 {
		return "", fmt.Errorf(
			"No public key files found in '%s'. Files need to end in '.key'.",
			publicKeyDir,
		)
	}

	entityList, err := utils.GetEntityList(keyFiles, "")
	if err != nil {
		return "", err
	}

	for _, param := range p {
		rendered, err := param.Render(entityList, previousParams)
		if err != nil {
			return "", err
		}
		out = out + rendered + "\n"
	}
	return out, nil
}

func (p Params) Process(dropSuffix bool, decode bool) (string, error) {
	out := ""
	for _, param := range p {
		processedParam, err := param.Process(dropSuffix, decode)
		if err != nil {
			return out, err
		}
		out = out + processedParam + "\n"
	}
	return out, nil
}

// Returns a string representation of the param.
// .ENC params are encrypted.
func (p *Param) Render(entityList openpgp.EntityList, previousParams Params) (string, error) {
	if !p.IsSecret {
		cli.DebugMsg("Rendering RAW param", p.Key)
		return p.Key + "=" + p.Value, nil
	}
	var previous *Param
	for _, prev := range previousParams {
		if prev.IsSecret && prev.Key == p.Key {
			previous = prev
			break
		}
	}
	var encrypted string
	if previous != nil && previous.Decrypted == p.Decrypted {
		cli.DebugMsg("Rendering unchanged ENC param", p.Key)
		encrypted = previous.Value
	} else {
		cli.DebugMsg("Rendering changed ENC param", p.Key)
		e, err := utils.Encrypt(p.Decrypted, entityList)
		if err != nil {
			return "", err
		}
		encrypted = e
	}
	return p.Key + ".ENC=" + encrypted, nil
}

// Returns a string representation in which all .ENC params are decrypted.
func (p *Param) Process(dropSuffix bool, decode bool) (string, error) {
	if !p.IsSecret {
		return p.Key + "=" + p.Value, nil
	}
	decrypted := p.Decrypted
	if decode {
		sDec, err := base64.StdEncoding.DecodeString(decrypted)
		if err != nil {
			return "", err
		}
		decrypted = string(sDec)
	}
	if dropSuffix {
		return p.Key + "=" + decrypted, nil
	}
	return p.Key + ".ENC=" + decrypted, nil
}
