package html

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ExtractCSS extracts all CSS styles from HTML content
func (h *HTML) ExtractCSS() []string {
	var cssContents []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "style" || (n.Data == "link" && getAttributeValue(n, "rel") == "stylesheet")) {
			if n.Data == "style" {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.TextNode {
						cssContents = append(cssContents, c.Data)
					}
				}
			} else if n.Data == "link" {
				// If link tag with stylesheet reference, handle if needed
				// For now, we're only considering inline styles
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(h.htmlNode)

	return cssContents
}

// getAttributeValue retrieves the value of the specified attribute from the node
func getAttributeValue(n *html.Node, attrName string) string {
	for _, attr := range n.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

// ExtractImageURLs extracts image URLs from CSS content
func (h *HTML) ExtractImageURLs(cssContents []string) []string {
	var imageUrls []string
	regex := regexp.MustCompile(`url\(['"]?(.*?)['"]?\)`)
	for _, css := range cssContents {
		matches := regex.FindAllStringSubmatch(css, -1)
		for _, match := range matches {
			if len(match) > 1 {
				imageUrls = append(imageUrls, match[1])
			}
		}
	}
	return imageUrls
}

// ExtractImageURLs extracts image URLs from CSS content
func (h *HTML) ExtractURLs() []string {
	var urlList []string
	urls := regexp.MustCompile(`url\((https?://[^\s]+)\)`).FindAllStringSubmatch(h.body, -1)

	// Download each referenced file if it hasn't been visited before
	for _, url := range urls {
		link := strings.Trim(url[1], "'\"")
		if link == "" {
			continue
		}
		urlList = append(urlList, link)
	}

	return urlList
}
