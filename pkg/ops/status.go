package ops

import (
	"fmt"

	"github.com/can3p/blg/pkg/store"
)

func OperationStatus(rootFolder string) error {
	cfg, err := store.Load(rootFolder)

	if err != nil {
		return err
	}

	folderState, err := store.ExamineFolder(cfg, rootFolder)

	if err != nil {
		return err
	}

	if len(folderState.New) == 0 {
		fmt.Println("No new files")
	} else {
		fmt.Printf("Found %d new files:\n\n", len(folderState.New))
		for _, ff := range folderState.New {
			fmt.Printf("  - %s\n", ff.FileName)
		}
		fmt.Println()
	}

	if len(folderState.Modified) == 0 {
		fmt.Println("No modified files")
	} else {
		fmt.Printf("Found %d modified files:\n\n", len(folderState.Modified))
		for _, ff := range folderState.Modified {
			fmt.Printf("  - %s:\n", ff.FileName)
		}
		fmt.Println()
	}

	if len(folderState.Deleted) == 0 {
		fmt.Println("No deleted files")
	} else {
		fmt.Printf("Found %d deleted files:\n\n", len(folderState.Deleted))
		for _, ff := range folderState.Deleted {
			fmt.Printf("  - %s:\n", ff.FileName)
		}
		//fmt.Println()
	}

	return nil
}
