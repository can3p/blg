package pwd

func getCmd(login, url string) []string {
	return []string{"secret-tool", "lookup", url + ":login", login}
}

func setCmd(login, url string) []string {
	return []string{"secret-tool", "store", "--label='" + url + "'", url + ":login", login}
}

func setPassword(login, url string) error {
	return runCmd(setCmd(login, url))
}
