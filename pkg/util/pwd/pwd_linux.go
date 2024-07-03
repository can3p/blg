package pwd

import "github.com/can3p/blg/pkg/util/exec"

func getCmd(login, url string) []string {
	return []string{"secret-tool", "lookup", url + ":login", login}
}

func setCmd(login, url string) []string {
	return []string{"secret-tool", "store", "--label='" + url + "'", url + ":login", login}
}

func setPassword(login, url string) error {
	_, err := exec.RunCmd(setCmd(login, url)...)

	return err
}
