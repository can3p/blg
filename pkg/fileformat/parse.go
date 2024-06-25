package fileformat

import (
	"regexp"
	"strings"
)

var headerRe = regexp.MustCompile(`^(\w+)\s*:\s*(.+)$`)

func ParseExportedPost(b []byte) (map[string]string, string, error) {
	lines := strings.Split(string(b), "\n")

	headers := map[string]string{}
	body := []string{}

	collectBody := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if collectBody {
			body = append(body, line)
			continue
		}

		if line == "" {
			continue
		}

		if matched := headerRe.FindStringSubmatch(line); matched != nil {
			headers[matched[1]] = matched[2]
		} else {
			body = append(body, line)
			collectBody = true
		}
	}

	return headers, strings.Join(body, "\n"), nil
}
