package utils

import (
	"bytes"
	"encoding/base64"
	"errors"
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

		filename := publicKeyDir + string(os.PathSeparator) + file.Name()
		keyringFileBuffer, _ := os.Open(filename)
		defer keyringFileBuffer.Close()
		l, err := openpgp.ReadArmoredKeyRing(keyringFileBuffer)
		if err != nil {
			return "", errors.New(
				fmt.Sprintf("Reading public key '%s' failed: %s", filename, err),
			)
		}
		entityList = append(entityList, l[0])
	}

	// Encrypt message using public keys
	buf := new(bytes.Buffer)
	w, err := openpgp.Encrypt(buf, entityList, nil, nil, nil)
	if err != nil {
		return "", errors.New(
			fmt.Sprintf("Encrypting '%s' failed: %s", secret, err),
		)
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
	if err != nil {
		return "", err
	}
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
		return "", errors.New(
			fmt.Sprintf("Reading private key '%s' failed: %s", privateKey, err),
		)
	}

	// Decode bas64-encoded string
	encrypted, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New(
			fmt.Sprintf("Decoding '%s' failed: %s", encoded, err),
		)
	}

	// Decrypt encrypted message
	buf := bytes.NewBuffer([]byte(encrypted))
	md, err := openpgp.ReadMessage(buf, entityList, nil, nil)
	if err != nil {
		return "", errors.New(
			fmt.Sprintf("Decrypting '%s' failed: %s", encrypted, err),
		)
	}
	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	return string(bytes), err
}
