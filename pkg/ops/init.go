package ops

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/can3p/blg/pkg/types"
	"github.com/can3p/blg/pkg/util/pwd"
	"github.com/pkg/errors"
)

func OperationInit(name, customHost, rootFolder string) error {
	alreadyDone, err := folderIsSetUp(rootFolder)

	if err != nil {
		return err
	}

	if alreadyDone {
		return errors.Errorf("The folder has already been initialized")
	}

	sd, ok := types.DefaultServiceRepo.Get(name)

	if !ok {
		return errors.Errorf("Unknown service name")
	}

	login := getLogin()

	if login == "" {
		return errors.Errorf("Login cannot be empty")
	}

	fmt.Println("h", sd.DefaultHost)
	fmt.Println("ch", customHost)

	_, err = pwd.GetAndSetPassword(login, cmp.Or(customHost, sd.DefaultHost))

	if err != nil {
		return err
	}

	newConfig := types.StoredConfig{
		Login:       login,
		ServiceName: sd.Name,
		CustomHost:  customHost,
	}

	return saveConfig(newConfig, rootFolder)
}

func getLogin() string {
	fmt.Print("Enter Your First Name: ")

	var login string

	// Taking input from user
	fmt.Scanln(&login)

	return strings.TrimSpace(login)
}

func configPath(folder string) string {
	return path.Join(folder, types.ConfigName)
}

func folderIsSetUp(root string) (bool, error) {
	fname := configPath(root)
	_, err := os.Stat(fname)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, err
}

func saveConfig(c types.StoredConfig, root string) error {
	b, err := json.MarshalIndent(c, "", "  ")

	if err != nil {
		return err
	}

	// silence errors in case folder already exists,
	// we care only about write file errors anyway
	_ = os.MkdirAll(root, 0755)

	return os.WriteFile(configPath(root), b, 0644)
}
