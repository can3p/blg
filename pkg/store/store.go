package store

import (
	"cmp"
	"encoding/json"
	"os"
	"path"

	"github.com/can3p/blg/pkg/types"
	"github.com/pkg/errors"
)

const ConfigName = "posts.json"

func Load(root string) (*types.Config, error) {
	configExists, err := FolderIsSetUp(root)

	if err != nil {
		return nil, err
	}

	if !configExists {
		return nil, errors.Errorf("Folder is not yet set up")
	}

	b, err := os.ReadFile(ConfigPath(root))

	if err != nil {
		return nil, err
	}

	var s types.StoredConfig

	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}

	sd, ok := types.DefaultServiceRepo.Get(s.ServiceName)

	if !ok {
		return nil, errors.Errorf("Cannot recognize service [%s] from the config", s.ServiceName)
	}

	return &types.Config{
		Stored:   s,
		RootPath: root,
		Host:     cmp.Or(s.CustomHost, sd.DefaultHost),
	}, nil

}

func ConfigPath(folder string) string {
	return path.Join(folder, ConfigName)
}

func FolderIsSetUp(root string) (bool, error) {
	fname := ConfigPath(root)
	_, err := os.Stat(fname)

	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, err
}

func SaveConfig(c types.StoredConfig, root string) error {
	b, err := json.MarshalIndent(c, "", "  ")

	if err != nil {
		return err
	}

	// silence errors in case folder already exists,
	// we care only about write file errors anyway
	_ = os.MkdirAll(root, 0755)

	return os.WriteFile(ConfigPath(root), b, 0644)
}
