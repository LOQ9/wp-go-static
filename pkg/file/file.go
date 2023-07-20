package file

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"wp-go-static/pkg/url"

	"github.com/gocolly/colly"
)

// NewCollector creates a new colly.Collector with some default settings
func NewCollector() *colly.Collector {
	c := colly.NewCollector()

	// Ignore SSL errors
	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

	return c
}

// HandleFile handles the file and returns the directory and file name
func HandleFile(r *colly.Response, filePath string) (string, string) {
	baseDir, fileName, err := url.ParsePath(r.Request.URL.String())
	if err != nil {
		fmt.Println(err)
		return "", ""
	}

	fileName, err = GetExtension(fileName, r.Headers.Get("Content-Type"))
	if err != nil {
		fmt.Println(err)
		return "", ""
	}

	if fileName == "" || fileName == "/" {
		fileName = "index.html"
	}

	dir := filepath.Join(".", filePath, baseDir)
	err = createDirectory(dir)
	if err != nil {
		fmt.Println(err)
		return "", ""
	}

	return dir, fileName
}

// createDirectory creates the directory if it does not exist
func createDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	}
	return nil
}

// SaveFile saves the response to a file
func SaveFile(r *colly.Response, dir string, fileName string) error {
	err := r.Save(filepath.Join(dir, fileName))
	if err != nil {
		return fmt.Errorf("error saving file: %v", err)
	}
	return nil
}

// GetExtension returns the extension of the file name or the extension of the content type
func GetExtension(fileName string, contentType string) (string, error) {
	ext := filepath.Ext(fileName)
	if ext == "" || ext == "." {
		metaExtension, err := mime.ExtensionsByType(contentType)
		if err != nil {
			return "", fmt.Errorf("error getting extension from content type: %v", err)
		}

		// Ensure that the extesion exists
		if len(metaExtension) == 0 {
			ext = metaExtension[0]
		} else {
			ext = ".html"
		}

		if ext == ".htm" {
			ext = ".html"
		}

		fileName = fmt.Sprintf("index%s", ext)
	}
	return fileName, nil
}
