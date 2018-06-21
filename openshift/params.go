package openshift

import (
	"encoding/base64"
	"github.com/michaelsauter/ocdiff/utils"
	"io/ioutil"
	"strings"
)

type ParamFromInput struct {
	Key      string
	IsSecret bool
	Value    string
}

type ParamFromFile struct {
	Key       string
	Value     string
	IsSecret  bool
	Decrypted string
}

type ParamsFromInput []*ParamFromInput
type ParamsFromFile []*ParamFromFile

func NewParamsFromInput(content string) ParamsFromInput {
	params := []*ParamFromInput{}
	text := strings.TrimSuffix(content, "\n")
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		pair := strings.SplitN(line, "=", 2)
		key := pair[0]
		value := pair[1]
		param := &ParamFromInput{}

		// If the key ends with .STRING, base64 encode the value and then
		// change the key to end with .ENC to trigger the next step.
		if strings.HasSuffix(key, ".STRING") {
			value = base64.StdEncoding.EncodeToString([]byte(value))
			key = strings.Replace(key, ".STRING", ".ENC", -1)
		}

		if strings.HasSuffix(key, ".ENC") {
			param.IsSecret = true
			param.Value = value
			param.Key = strings.Replace(key, ".ENC", "", -1)
		} else {
			param.IsSecret = false
			param.Value = value
			param.Key = key
		}

		params = append(params, param)
	}

	return params
}

func NewParamsFromFile(filename string, privateKey string) (ParamsFromFile, error) {
	params := []*ParamFromFile{}
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return params, err
	}
	text := strings.TrimSuffix(string(content), "\n")
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		pair := strings.SplitN(line, "=", 2)
		key := pair[0]
		value := pair[1]
		param := &ParamFromFile{}

		if strings.HasSuffix(key, ".ENC") {
			param.IsSecret = true
			param.Value = value
			decrypted, _ := utils.Decrypt(value, privateKey)
			param.Decrypted = decrypted
			param.Key = strings.Replace(key, ".ENC", "", -1)
		} else {
			param.IsSecret = false
			param.Value = value
			param.Key = key
		}

		params = append(params, param)
	}

	return params, nil
}

func (p ParamsFromInput) String() string {
	out := ""
	for _, param := range p {
		out = out + param.Key + "=" + param.Value + "\n"
	}
	return out
}

func (p ParamsFromFile) String() string {
	out := ""
	for _, param := range p {
		out = out + param.Key + "=" + param.Value + "\n"
	}
	return out
}

// Encrypt params and create string from them
func (p ParamsFromInput) Render(publicKeyDir string, previousParams ParamsFromFile) string {
	out := ""
	for _, param := range p {
		out = out + param.Render(publicKeyDir, previousParams) + "\n"
	}
	return out
}

func (p ParamsFromFile) Process(dropSuffix bool, decode bool) string {
	out := ""
	for _, param := range p {
		out = out + param.Process(dropSuffix, decode) + "\n"
	}
	return out
}

// Returns a string representation of the param.
// .ENC params are encrypted.
func (p *ParamFromInput) Render(publicKeyDir string, previousParams ParamsFromFile) string {
	if !p.IsSecret {
		return p.Key + "=" + p.Value
	}
	var previous *ParamFromFile
	for _, prev := range previousParams {
		if prev.IsSecret && prev.Key == p.Key {
			previous = prev
			break
		}
	}
	var encrypted string
	if previous != nil && previous.Decrypted == p.Value {
		encrypted = previous.Value
	} else {
		encrypted, _ = utils.Encrypt(p.Value, publicKeyDir)
	}
	return p.Key + ".ENC=" + encrypted
}

// Returns a string representation in which all .ENC params are decrypted.
func (p *ParamFromFile) Process(dropSuffix bool, decode bool) string {
	if !p.IsSecret {
		return p.Key + "=" + p.Value
	}
	decrypted := p.Decrypted
	if decode {
		sDec, err := base64.StdEncoding.DecodeString(decrypted)
		if err != nil {

		}
		decrypted = string(sDec)
	}
	if dropSuffix {
		return p.Key + "=" + decrypted
	}
	return p.Key + ".ENC=" + decrypted
}
