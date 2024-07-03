package exec

import (
	"os"
	"os/exec"
	"strings"
)

func RunCmd(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func EditFile(args ...string) error {
	editorPath := os.Getenv("EDITOR")
	if editorPath == "" {
		editorPath = "vim" // coz it rocks
	}

	cmd := exec.Command(editorPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
