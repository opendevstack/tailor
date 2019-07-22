package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/opendevstack/tailor/cli"
	"github.com/opendevstack/tailor/openshift"
	"github.com/opendevstack/tailor/utils"
)

// GenerateKey generates a GPG key using specified email (and optionally name).
func GenerateKey(globalOptions *cli.GlobalOptions, email, name string) error {
	emailParts := strings.Split(email, "@")
	if len(name) == 0 {
		name = emailParts[0]
	}
	entity, err := utils.CreateEntity(name, email)
	if err != nil {
		return fmt.Errorf("Failed to generate keypair: %s", err)
	}
	publicKeyFilename := strings.Replace(emailParts[0], ".", "-", -1) + ".key"
	err = utils.PrintPublicKey(entity, publicKeyFilename)
	if err != nil {
		return err
	}
	fmt.Printf("Public Key written to %s. This file can be committed.\n", publicKeyFilename)
	privateKeyFilename := globalOptions.PrivateKey
	err = utils.PrintPrivateKey(entity, privateKeyFilename)
	if err != nil {
		return err
	}
	fmt.Printf("Private Key written to %s. This file MUST NOT be committed.\n", privateKeyFilename)
	return nil
}

// Reveal prints the clear-text of an encrypted file to STDOUT.
func Reveal(globalOptions *cli.GlobalOptions, filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("'%s' does not exist", filename)
	}
	encryptedContent, err := utils.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Could not read file: %s", err)
	}
	decryptedContent, err := openshift.DecryptedParams(
		encryptedContent,
		globalOptions.PrivateKey,
		globalOptions.Passphrase,
	)
	if err != nil {
		return fmt.Errorf("Could not decrypt file: %s", err)
	}
	fmt.Println(decryptedContent)
	return nil
}

// ReEncrypt decrypts given file(s) and encrypts all params again.
// This allows to share the secrets with a new keypair.
func ReEncrypt(globalOptions *cli.GlobalOptions, filename string) error {
	if len(filename) > 0 {
		err := reEncrypt(filename, globalOptions.PrivateKey, globalOptions.Passphrase, globalOptions.PublicKeyDir)
		if err != nil {
			return err
		}
	} else {
		for _, paramDir := range globalOptions.ParamDirs {
			files, err := ioutil.ReadDir(paramDir)
			if err != nil {
				return err
			}
			filePattern := ".*\\.env.enc$"
			re := regexp.MustCompile(filePattern)
			for _, file := range files {
				matched := re.MatchString(file.Name())
				if !matched {
					continue
				}
				filename := paramDir + string(os.PathSeparator) + file.Name()
				err := reEncrypt(filename, globalOptions.PrivateKey, globalOptions.Passphrase, globalOptions.PublicKeyDir)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Edit opens given filen in cleartext in $EDITOR, then encrypts the content on save.
func Edit(globalOptions *cli.GlobalOptions, filename string) error {
	encryptedContent, err := utils.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			cli.DebugMsg(filename, "does not exist, creating empty file")
		} else {
			return fmt.Errorf("Could not read file: %s", err)
		}
	}

	cleartextContent, err := openshift.DecryptedParams(
		encryptedContent,
		globalOptions.PrivateKey,
		globalOptions.Passphrase,
	)
	if err != nil {
		return fmt.Errorf("Could not decrypt file: %s", err)
	}

	editedContent, err := cli.EditEnvFile(cleartextContent)
	if err != nil {
		return fmt.Errorf("Could not edit file: %s", err)
	}

	err = writeEncryptedContent(
		filename,
		editedContent,
		encryptedContent,
		globalOptions.PrivateKey,
		globalOptions.Passphrase,
		globalOptions.PublicKeyDir,
	)
	if err != nil {
		return fmt.Errorf("Could not write file: %s", err)
	}
	return nil
}

func reEncrypt(filename, privateKey, passphrase, publicKeyDir string) error {
	encryptedContent, err := utils.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Could not read file: %s", err)
	}

	cleartextContent, err := openshift.DecryptedParams(
		encryptedContent,
		privateKey,
		passphrase,
	)
	if err != nil {
		return fmt.Errorf("Could not decrypt file: %s", err)
	}

	return writeEncryptedContent(
		filename,
		cleartextContent,
		"", // empty because all values should be re-encrypted
		privateKey,
		passphrase,
		publicKeyDir,
	)
}

func writeEncryptedContent(filename, newContent, previousContent, privateKey, passphrase, publicKeyDir string) error {
	updatedContent, err := openshift.EncryptedParams(
		newContent,
		previousContent,
		publicKeyDir,
		privateKey,
		passphrase,
	)
	if err != nil {
		return fmt.Errorf("Could not encrypt content: %s", err)
	}

	err = ioutil.WriteFile(filename, []byte(updatedContent), 0644)
	if err != nil {
		return fmt.Errorf("Could not write file: %s", err)
	}
	return nil
}
