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
	New      []*types.PostMeta
	Modified []*types.PostMeta
	Deleted  []*types.PostMeta
}

func ExamineFolder(cfg *types.Config, rootFolder string) (*FolderState, error) {
	existing := lo.KeyBy(cfg.Stored.RemotePosts, func(p *types.PostMeta) string {
		return p.FileName
	})

	mdFiles, err := CollecMdPaths(rootFolder)

	if err != nil {
		return nil, err
	}

	newFiles := []*types.PostMeta{}
	modifiedFiles := []*types.PostMeta{}
	deletedFiles := []*types.PostMeta{}

	for _, md := range mdFiles {
		found, ok := existing[md]

		hash, err := CalcHash(md)

		if err != nil {
			return nil, err
		}

		if !ok {
			newFiles = append(newFiles, &types.PostMeta{
				FileName: md,
				Hash:     hash,
			})

			continue
		}

		delete(existing, md)

		if hash == found.Hash {
			continue
		}

		modifiedFiles = append(modifiedFiles, found)
	}

	for _, existing := range existing {
		deletedFiles = append(deletedFiles, existing)
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
			out = append(out, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}
