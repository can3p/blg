package pwd

import (
	"github.com/can3p/blg/pkg/util/exec"
	"github.com/pkg/errors"
)

func getCmd(login, url string) []string {
	return []string{"security", "find-internet-password", "-a", login, "-s", url, "-w"}
}

func setCmd(login, url, password string) []string {
	return []string{"security", "add-internet-password", "-a", login, "-s", url, "-w", password}
}

func setPassword(login, url string) error {
	password, err := readPassword()

	if err != nil {
		return err
	}

	if password == "" {
		return errors.Errorf("Password cannot be empty")
	}

	_, err = exec.RunCmd(setCmd(login, url, password)...)

	return err
}
