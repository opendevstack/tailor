package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"io/ioutil"
	"os"
	"regexp"
)

// Encrypts secret with all public keys and base64-encodes the result.
func Encrypt(secret string, publicKeyDir string) (string, error) {
	// Read public keys
	entityList := []*openpgp.Entity{}
	files, err := ioutil.ReadDir(publicKeyDir)
	if err != nil {
		return "", err
	}
	filePattern := ".*\\.key"
	for _, file := range files {
		matched, _ := regexp.MatchString(filePattern, file.Name())
		if !matched {
			continue
		}

		keyringFileBuffer, _ := os.Open(publicKeyDir + string(os.PathSeparator) + file.Name())
		defer keyringFileBuffer.Close()
		l, err := openpgp.ReadArmoredKeyRing(keyringFileBuffer)
		if err != nil {
			fmt.Println("Could not read")
			return "", err
		}
		entityList = append(entityList, l[0])
	}

	// Encrypt message using public keys
	buf := new(bytes.Buffer)
	w, err := openpgp.Encrypt(buf, entityList, nil, nil, nil)
	if err != nil {
		return "", err
	}
	_, err = w.Write([]byte(secret))
	if err != nil {
		return "", err
	}
	err = w.Close()
	if err != nil {
		return "", err
	}

	// Return as base64 encoded string
	bytes, err := ioutil.ReadAll(buf)
	str := base64.StdEncoding.EncodeToString(bytes)
	return str, nil
}

// Decrypts the base64-encoded string end decrypts with the private key.
func Decrypt(encoded string, privateKey string) (string, error) {
	// Read private key
	keyringFileBuffer, _ := os.Open(privateKey)
	defer keyringFileBuffer.Close()
	entityList, err := openpgp.ReadArmoredKeyRing(keyringFileBuffer)
	if err != nil {
		return "", err
	}

	// Decode bas64-encoded string
	encrypted, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	// Decrypt encrypted message
	buf := bytes.NewBuffer([]byte(encrypted))
	md, err := openpgp.ReadMessage(buf, entityList, nil, nil)
	if err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	return string(bytes), nil
}
