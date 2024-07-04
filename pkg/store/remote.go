package store

import (
	"encoding/json"
	"os"
	"path"

	"github.com/can3p/blg/pkg/types"
	"github.com/pkg/errors"
)

const RemoteConfigName = "remote.json"

func LoadRemotePosts(root string) (*types.RemotePosts, error) {
	configExists, err := FolderIsSetUp(root)

	if err != nil {
		return nil, err
	}

	if !configExists {
		return nil, errors.Errorf("Folder is not yet set up")
	}

	_, err = os.Stat(RemoteConfigPath(root))

	if err != nil {
		if os.IsNotExist(err) {
			return &types.RemotePosts{
				Version: 1,
			}, nil
		}

		return nil, err
	}

	b, err := os.ReadFile(RemoteConfigPath(root))

	if err != nil {
		return nil, err
	}

	var s types.RemotePosts

	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

func RemoteConfigPath(folder string) string {
	return path.Join(folder, RemoteConfigName)
}

func SaveRemotePosts(c types.RemotePosts, root string) error {
	b, err := json.MarshalIndent(c, "", "  ")

	if err != nil {
		return err
	}

	// silence errors in case folder already exists,
	// we care only about write file errors anyway
	_ = os.MkdirAll(root, 0755)

	return os.WriteFile(RemoteConfigPath(root), b, 0644)
}
