package pwd

import (
	"bytes"
	"fmt"
	"strings"
	"syscall"

	"github.com/can3p/blg/pkg/util/exec"
	"github.com/pkg/errors"
	"golang.org/x/term"
)

func GetPassword(login, url string) (string, error) {
	password, err := exec.RunCmd(getCmd(login, url)...)

	if err != nil {
		// max os x: exit status 44 only means that there is no password in the key chain
		if strings.Contains(err.Error(), "exit status 44") {
			return "", nil
		}
	}

	return strings.TrimSpace(password), err
}

func GetAndSetPassword(login, url string) (string, error) {
	password, err := GetPassword(login, url)

	if err != nil {
		return "", errors.Wrapf(err, "failed to read password")
	}

	if password == "" {
		err := setPassword(login, url)

		if err != nil {
			return "", err
		}

		return GetPassword(login, url)
	}

	return password, nil
}

func readPassword() (string, error) {
	fmt.Printf("Your password/auth key: ")

	pwd, err := term.ReadPassword(syscall.Stdin)

	if err != nil {
		return "", errors.Wrapf(err, "failed to read password")
	}

	return string(bytes.TrimSpace(pwd)), nil
}
