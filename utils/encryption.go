package utils

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"io/ioutil"
	"os"
)

// Assembles entity list from keys in given files
func GetEntityList(keys []string) (openpgp.EntityList, error) {
	entityList := openpgp.EntityList{}
	for _, filename := range keys {
		keyringFileBuffer, _ := os.Open(filename)
		defer keyringFileBuffer.Close()
		l, err := openpgp.ReadArmoredKeyRing(keyringFileBuffer)
		if err != nil {
			return entityList, errors.New(
				fmt.Sprintf("Reading key '%s' failed: %s", filename, err),
			)
		}
		entityList = append(entityList, l[0])
	}
	return entityList, nil
}

// Encrypts secret with all public keys and base64-encodes the result.
func Encrypt(secret string, entityList openpgp.EntityList) (string, error) {
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
func Decrypt(encoded string, entityList openpgp.EntityList) (string, error) {
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
