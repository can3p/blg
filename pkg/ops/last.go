package ops

import (
	"path/filepath"

	"github.com/can3p/blg/pkg/store"
	"github.com/can3p/blg/pkg/util/exec"
	"github.com/pkg/errors"
)

func OperationLast(root string) error {
	config, err := store.Load(root)

	if err != nil {
		return err
	}

	if len(config.Stored.RemotePosts) == 0 {
		return errors.Errorf("No remote posts found")
	}

	last := config.Stored.RemotePosts[len(config.Stored.RemotePosts)-1]

	fullPath := filepath.Join(root, last.FileName)
	return exec.EditFile(fullPath)
}
