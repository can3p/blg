package ops

import (
	"fmt"

	"github.com/can3p/blg/pkg/store"
	"github.com/can3p/blg/pkg/types"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

func OperationUrl(rootFolder string, fname string) error {
	config, err := store.Load(rootFolder)

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

	meta, ok := lo.Find(config.Stored.RemotePosts, func(p *types.PostMeta) bool {
		return p.FileName == fname
	})

	if !ok {
		return errors.Errorf("File %s does not correspond to any post", fname)
	}

	fmt.Println(service.PostURL(meta.RemoteID))

	return nil
}
