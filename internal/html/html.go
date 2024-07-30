package html

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

type HTML struct {
	body     string
	htmlNode *html.Node
}

func NewHTML(body string) *HTML {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		fmt.Println("Error parsing HTML:", err)
		return nil
	}

	return &HTML{
		body:     body,
		htmlNode: doc,
	}
}
