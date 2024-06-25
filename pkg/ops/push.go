package ops

import (
	"os"
	"path/filepath"

	"github.com/can3p/blg/pkg/fileformat"
	"github.com/can3p/blg/pkg/store"
	"github.com/can3p/blg/pkg/types"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

func OperationPush(rootFolder string, singleFname string) error {
	cfg, err := store.Load(rootFolder)

	if err != nil {
		return err
	}

	folderState, err := store.ExamineFolder(cfg, rootFolder)

	if err != nil {
		return err
	}

	if singleFname != "" {
		if meta, ok := folderState.New[singleFname]; ok {
			return performPush(rootFolder, meta, PushOpCreate)
		} else if meta, ok := folderState.Modified[singleFname]; ok {
			return performPush(rootFolder, meta, PushOpUpdate)
		} else if meta, ok := folderState.Deleted[singleFname]; ok {
			return performPush(rootFolder, meta, PushOpDelete)
		} else {
			return errors.Errorf("File %s is not modified, nothing to do", singleFname)
		}
	}

	for _, meta := range folderState.New {
		if err := performPush(rootFolder, meta, PushOpCreate); err != nil {
			return err
		}
	}

	for _, meta := range folderState.Modified {
		if err := performPush(rootFolder, meta, PushOpUpdate); err != nil {
			return err
		}
	}

	for _, meta := range folderState.Deleted {
		if err := performPush(rootFolder, meta, PushOpDelete); err != nil {
			return err
		}
	}

	return nil
}

type PushOp int

const (
	PushOpCreate PushOp = iota
	PushOpUpdate
	PushOpDelete
)

func preCreateUpdate(root string, config *types.Config, service types.Service, fileName string) (*types.Post, error) {
	contents, err := os.ReadFile(fileName)

	if err != nil {
		return nil, err
	}

	// we need to handle images at this point
	fields, body, err := fileformat.ParseExportedPost(contents)

	if err != nil {
		return nil, err
	}

	prepared, images, err := service.PreparePost(fields, body)

	if err != nil {
		return nil, err
	}

	replaceMap := map[string]string{}

	for _, imFname := range images {
		// @TODO: hash check should happen there
		if imgMeta, ok := lo.Find(config.Stored.RemoteImages, func(im *types.ImageMeta) bool {
			return im.FileName == imFname
		}); ok {
			replaceMap[imFname] = imgMeta.RemoteID
			continue
		}

		remoteImageID, err := service.UploadImage(filepath.Join(root, imFname))

		if err != nil {
			return nil, err
		}

		newMeta := &types.ImageMeta{
			FileName: imFname,
			RemoteID: remoteImageID,
		}

		config.Stored.RemoteImages = append(config.Stored.RemoteImages, newMeta)

		if _, err := os.Stat(filepath.Join(root, imFname)); err == nil {
			replaceMap[imFname] = remoteImageID
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	if err := prepared.Body.ReplaceImages(replaceMap); err != nil {
		return nil, err
	}

	return prepared, nil
}

func performPush(root string, meta *types.PostMeta, op PushOp) error {
	relFileName := meta.FileName
	fileName := filepath.Join(root, relFileName)
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

	switch op {
	case PushOpCreate:
		prepared, err := preCreateUpdate(root, config, service, fileName)

		if err != nil {
			return err
		}

		remoteID, err := service.Create(prepared)

		if err != nil {
			return err
		}

		hash, err := store.CalcHash(fileName)

		if err != nil {
			return err
		}

		newMeta := &types.PostMeta{
			FileName: fileName,
			RemoteID: remoteID,
			Hash:     hash,
		}

		config.Stored.RemotePosts = append(config.Stored.RemotePosts, newMeta)
	case PushOpUpdate:
		prepared, err := preCreateUpdate(root, config, service, fileName)

		if err != nil {
			return err
		}

		err = service.Update(meta.RemoteID, prepared)

		if err != nil {
			return err
		}

		hash, err := store.CalcHash(fileName)

		if err != nil {
			return err
		}

		meta, ok := lo.Find(config.Stored.RemotePosts, func(p *types.PostMeta) bool {
			return p.FileName == relFileName
		})

		if !ok {
			return errors.Errorf("Updated a post that does not exist in the database")
		}

		meta.Hash = hash
	case PushOpDelete:
		err = service.Delete(meta.RemoteID)

		if err != nil {
			return err
		}

		config.Stored.RemotePosts = lo.Reject(config.Stored.RemotePosts, func(p *types.PostMeta, idx int) bool {
			return p.FileName == relFileName
		})
	}

	return store.SaveConfig(config.Stored, root)
}
