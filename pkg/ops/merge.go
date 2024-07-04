package ops

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/can3p/blg/pkg/store"
	"github.com/can3p/blg/pkg/types"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

func OperationMerge(root string) error {
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

	remote, err := store.LoadRemotePosts(root)

	if err != nil {
		return err
	}

	if len(remote.Posts) == 0 {
		return nil
	}

	state, err := store.ExamineRemote(config, remote)

	if err != nil {
		return err
	}

	if len(state.New) == 0 && len(state.Modified) == 0 {
		return nil
	}

	for _, p := range state.New {
		fname, b, err := service.FormatRemotePost(p)

		if err != nil {
			return err
		}

		fullPath := filepath.Join(root, fname)

		if err := os.WriteFile(fullPath, b, 0666); err != nil {
			return err
		}

		newMeta := &types.PostMeta{
			FileName: fname,
			RemoteID: p.ID,
			Hash:     fmt.Sprintf("%x", sha256.Sum256(b)),
			PushedAt: p.UpdatedAt,
		}

		config.Stored.RemotePosts = append(config.Stored.RemotePosts, newMeta)
	}

	if len(state.Modified) > 0 {
		local := lo.KeyBy(config.Stored.RemotePosts, func(r *types.PostMeta) string {
			return r.FileName
		})

		for _, p := range state.Modified {
			_, b, err := service.FormatRemotePost(&p.RemotePost)

			if err != nil {
				return err
			}

			fullPath := filepath.Join(root, p.Fname)

			if err := os.WriteFile(fullPath, b, 0666); err != nil {
				return err
			}

			meta := local[p.Fname]
			meta.Hash = fmt.Sprintf("%x", sha256.Sum256(b))
			meta.PushedAt = p.UpdatedAt
		}
	}

	return store.SaveConfig(config.Stored, root)
}
