package store

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/can3p/blg/pkg/types"
	"github.com/samber/lo"
)

type FolderState struct {
	New      map[string]*types.PostMeta
	Modified map[string]*types.PostMeta
	Deleted  map[string]*types.PostMeta
}

func ExamineFolder(cfg *types.Config, rootFolder string) (*FolderState, error) {
	existing := lo.KeyBy(cfg.Stored.RemotePosts, func(p *types.PostMeta) string {
		return p.FileName
	})

	mdFiles, err := CollecMdPaths(rootFolder)

	if err != nil {
		return nil, err
	}

	newFiles := map[string]*types.PostMeta{}
	modifiedFiles := map[string]*types.PostMeta{}
	deletedFiles := map[string]*types.PostMeta{}

	for _, md := range mdFiles {
		found, ok := existing[md]

		hash, err := CalcHash(filepath.Join(rootFolder, md))

		if err != nil {
			return nil, err
		}

		if !ok {
			newFiles[md] = &types.PostMeta{
				FileName: md,
				Hash:     hash,
			}

			continue
		}

		delete(existing, md)

		if hash == found.Hash {
			continue
		}

		modifiedFiles[found.FileName] = found
	}

	for _, existing := range existing {
		deletedFiles[existing.FileName] = existing
	}

	return &FolderState{
		New:      newFiles,
		Modified: modifiedFiles,
		Deleted:  deletedFiles,
	}, nil
}

func CalcHash(p string) (string, error) {
	b, err := os.ReadFile(p)

	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(b)

	return fmt.Sprintf("%x", hash), nil
}

func CollecMdPaths(rootFolder string) ([]string, error) {
	out := []string{}

	err := filepath.WalkDir(rootFolder, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".md") {
			relPath, err := filepath.Rel(rootFolder, path)

			if err != nil {
				return err
			}

			out = append(out, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}

type RemotePostWithLocal struct {
	types.RemotePost
	Fname string
}

type RemoteState struct {
	New      []*types.RemotePost
	Modified []*RemotePostWithLocal
}

func ExamineRemote(cfg *types.Config, remote *types.RemotePosts) (*RemoteState, error) {
	existing := lo.KeyBy(cfg.Stored.RemotePosts, func(p *types.PostMeta) string {
		return p.RemoteID
	})

	newFiles := []*types.RemotePost{}
	modifiedFiles := []*RemotePostWithLocal{}

	for _, p := range remote.Posts {
		local, ok := existing[p.ID]

		switch {
		case !ok:
			newFiles = append(newFiles, p)
		case p.UpdatedAt > local.PushedAt:
			modifiedFiles = append(modifiedFiles, &RemotePostWithLocal{
				RemotePost: *p,
				Fname:      local.FileName,
			})
		}
	}

	return &RemoteState{
		New:      newFiles,
		Modified: modifiedFiles,
	}, nil
}
