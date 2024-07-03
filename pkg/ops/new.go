package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/can3p/blg/pkg/store"
	"github.com/can3p/blg/pkg/types"
	"github.com/can3p/blg/pkg/util/exec"
	"github.com/pkg/errors"
)

func OperationNew(root, name string, open bool) error {
	config, err := store.Load(root)

	if err != nil {
		return err
	}

	sdef, ok := types.DefaultServiceRepo.Get(config.Stored.ServiceName)

	if !ok {
		return errors.Errorf("Service [%s] not found in the definitions", config.Stored.ServiceName)
	}

	service, err := sdef.GetService(config)

	if err != nil {
		return err
	}

	fname := fmt.Sprintf("%s-%s.md", time.Now().Format("2006-01-02"), name)
	content := service.NewPostTemplate(name)
	fullPath := filepath.Join(root, fname)

	err = os.WriteFile(fullPath, []byte(content), 0666)

	if err != nil {
		return err
	}

	fmt.Println(fullPath)

	if open {
		return exec.EditFile(fullPath)
	}

	return nil
}
