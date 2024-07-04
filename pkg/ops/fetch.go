package ops

import (
	"os"
	"path/filepath"
	"time"

	"github.com/can3p/blg/pkg/store"
	"github.com/can3p/blg/pkg/types"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

const RemoteImagesFolder = "remote_images"

func OperationFetch(root string) error {
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

	storedRemote, err := store.LoadRemotePosts(root)

	if err != nil {
		return err
	}

	newTs := time.Now().Unix()
	newRemote, imageIDs, err := service.FetchPosts(storedRemote.LastTS)

	if err != nil {
		return err
	}

	indMap := map[string]int{}

	for idx, p := range storedRemote.Posts {
		indMap[p.ID] = idx
	}

	for _, p := range newRemote {
		if idx, ok := indMap[p.ID]; ok {
			if storedRemote.Posts[idx].Hash != p.Hash {
				storedRemote.Posts[idx] = p
			}
			continue
		}

		storedRemote.Posts = append(storedRemote.Posts, p)
	}

	storedRemote.LastTS = newTs

	existingImgs := map[string]*types.ImageMeta{}

	for _, im := range config.Stored.RemoteImages {
		existingImgs[im.RemoteID] = im
	}

	imageIDs = lo.Filter(imageIDs, func(imageID string, idx int) bool {
		_, ok := existingImgs[imageID]

		return !ok
	})

	if len(imageIDs) > 0 {
		imgPath := filepath.Join(root, RemoteImagesFolder)

		if err := os.MkdirAll(imgPath, 0755); err != nil {
			return err
		}

		for _, imageID := range imageIDs {
			b, err := service.DownloadImage(imageID)

			if err != nil {
				return err
			}

			fname := filepath.Join(imgPath, imageID)

			if err := os.WriteFile(fname, b, 0666); err != nil {
				return err
			}

			newMeta := &types.ImageMeta{
				FileName: filepath.Join(RemoteImagesFolder, imageID),
				RemoteID: imageID,
			}

			config.Stored.RemoteImages = append(config.Stored.RemoteImages, newMeta)
		}

		if err := store.SaveConfig(config.Stored, root); err != nil {
			return err
		}
	}

	return store.SaveRemotePosts(*storedRemote, root)
}
