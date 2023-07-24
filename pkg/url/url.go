package url

import (
	"fmt"
	"net/url"
	"path/filepath"
)

// ParsePath parses the URL and returns the base directory and file name
func ParsePath(urlString string) (string, string, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return "", "", fmt.Errorf("error parsing URL: %v", err)
	}
	path := u.Path
	baseDir := filepath.Dir(path)
	fileName := filepath.Base(path)
	return baseDir, fileName, nil
}
