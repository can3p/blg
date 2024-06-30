package ops

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/can3p/blg/pkg/store"
	"github.com/can3p/blg/pkg/types"
	"github.com/can3p/blg/pkg/util/pwd"
	"github.com/pkg/errors"
)

func OperationInit(name, customHost, rootFolder string) error {
	alreadyDone, err := store.FolderIsSetUp(rootFolder)

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

	_, err = pwd.GetAndSetPassword(login, cmp.Or(customHost, sd.DefaultHost))

	if err != nil {
		return err
	}

	newConfig := types.StoredConfig{
		Version:     1,
		Login:       login,
		ServiceName: sd.Name,
		CustomHost:  customHost,
	}

	return store.SaveConfig(newConfig, rootFolder)
}

func getLogin() string {
	fmt.Print("Enter Your Login Name: ")

	var login string

	// Taking input from user
	fmt.Scanln(&login)

	return strings.TrimSpace(login)
}
