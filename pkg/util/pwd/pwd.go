package pwd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/term"
)

func GetPassword(login, url string) (string, error) {
	password, err := runCmd(getCmd(login, url)...)

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

func runCmd(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
