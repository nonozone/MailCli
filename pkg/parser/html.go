package parser

import (
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func cleanHTML(input string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(input))
	if err != nil {
		return "", err
	}

	doc.Find("style,script,svg,textarea").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		for _, node := range s.Nodes {
			filterAttrs(node)
		}
	})

	htmlStr, err := doc.Find("body").Html()
	if err == nil && strings.TrimSpace(htmlStr) != "" {
		return htmlStr, nil
	}

	full, err := doc.Html()
	if err != nil {
		return "", err
	}
	return full, nil
}

func filterAttrs(node *html.Node) {
	if node == nil {
		return
	}

	attrs := node.Attr[:0]
	for _, attr := range node.Attr {
		switch attr.Key {
		case "href", "src", "alt", "title":
			attrs = append(attrs, attr)
		}
	}
	node.Attr = attrs
}

func htmlToMarkdown(input string) (string, error) {
	converter := md.NewConverter("", true, nil)
	output, err := converter.ConvertString(input)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}
