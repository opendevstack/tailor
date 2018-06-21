package openshift

import (
	"encoding/base64"
	"github.com/michaelsauter/ocdiff/utils"
	"io/ioutil"
	"strings"
)

type Param struct {
	Key       string
	Value     string
	IsSecret  bool
	Decrypted string
}

type Params []*Param

func NewParams(content string, privateKey string) (Params, error) {
	params := []*Param{}
	text := strings.TrimSuffix(content, "\n")
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		pair := strings.SplitN(line, "=", 2)
		key := pair[0]
		value := pair[1]
		param := &Param{}

		// If the key ends with .STRING, base64 encode the value and then
		// change the key to end with .ENC to trigger the next step.
		if strings.HasSuffix(key, ".STRING") {
			value = base64.StdEncoding.EncodeToString([]byte(value))
			key = strings.Replace(key, ".STRING", ".ENC", -1)
		}

		if strings.HasSuffix(key, ".ENC") {
			param.IsSecret = true
			param.Key = strings.Replace(key, ".ENC", "", -1)
			if len(privateKey) > 0 {
				param.Value = value
				decrypted, err := utils.Decrypt(value, privateKey)
				if err != nil {
					return params, err
				}
				param.Decrypted = decrypted
			} else {
				param.Decrypted = value
			}
		} else {
			param.IsSecret = false
			param.Value = value
			param.Key = key
		}

		params = append(params, param)
	}

	return params, nil
}

func NewParamsFromInput(content string) (Params, error) {
	return NewParams(content, "")
}

func NewParamsFromFile(filename string, privateKey string) (Params, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NewParams(string(bytes), privateKey)
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
func (p Params) Render(publicKeyDir string, previousParams Params) string {
	out := ""
	for _, param := range p {
		out = out + param.Render(publicKeyDir, previousParams) + "\n"
	}
	return out
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
func (p *Param) Render(publicKeyDir string, previousParams Params) string {
	if !p.IsSecret {
		return p.Key + "=" + p.Value
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
		encrypted = previous.Value
	} else {
		encrypted, _ = utils.Encrypt(p.Decrypted, publicKeyDir)
	}
	return p.Key + ".ENC=" + encrypted
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
