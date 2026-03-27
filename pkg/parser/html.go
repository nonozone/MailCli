package parser

import (
	stdhtml "html"
	"math"
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

	doc.Find("style,script,svg,textarea,noscript,template,nav,footer,form").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})
	pruneLikelyNoise(doc.Selection)
	replaceCIDImages(doc.Selection)

	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		for _, node := range s.Nodes {
			filterAttrs(node)
		}
	})

	if htmlStr, ok := selectPrimaryHTML(doc); ok {
		return htmlStr, nil
	}

	full, err := doc.Html()
	if err != nil {
		return "", err
	}
	return full, nil
}

func selectPrimaryHTML(doc *goquery.Document) (string, bool) {
	for _, selector := range []string{"main", "article", `[role="main"]`} {
		selection := doc.Find(selector).First()
		if selection.Length() == 0 {
			continue
		}

		htmlStr, err := selection.Html()
		if err == nil && strings.TrimSpace(htmlStr) != "" {
			return htmlStr, true
		}
	}

	body := doc.Find("body").First()
	if body.Length() > 0 {
		if htmlStr, ok := selectHeuristicBodyRoot(body); ok {
			return htmlStr, true
		}

		body.ChildrenFiltered("header").Each(func(i int, s *goquery.Selection) {
			s.Remove()
		})
		htmlStr, err := body.Html()
		if err == nil && strings.TrimSpace(htmlStr) != "" {
			return htmlStr, true
		}
	}

	return "", false
}

func pruneLikelyNoise(root *goquery.Selection) {
	if root == nil {
		return
	}

	root.Find(`[class*="preheader"],[id*="preheader"]`).Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	root.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if !isLikelyNoiseLink(s) {
			return
		}

		container := closestNoiseContainer(s)
		if container != nil {
			container.Remove()
			return
		}
		s.Remove()
	})
}

func replaceCIDImages(root *goquery.Selection) {
	if root == nil {
		return
	}

	root.Find("img").Each(func(i int, s *goquery.Selection) {
		src := strings.TrimSpace(getAttr(s, "src"))
		if !strings.HasPrefix(strings.ToLower(src), "cid:") {
			return
		}

		alt := strings.TrimSpace(getAttr(s, "alt"))
		if alt == "" {
			s.Remove()
			return
		}

		_ = s.ReplaceWithHtml("<p>" + stdhtml.EscapeString(alt) + "</p>")
	})
}

func selectHeuristicBodyRoot(body *goquery.Selection) (string, bool) {
	children := body.Children()
	if children.Length() == 0 {
		return "", false
	}

	type candidate struct {
		selection *goquery.Selection
		score     float64
	}

	var best candidate
	second := -1.0

	children.Each(func(i int, s *goquery.Selection) {
		score := scorePrimaryCandidate(s)
		if score <= 0 {
			return
		}
		if score > best.score {
			second = best.score
			best = candidate{selection: s, score: score}
			return
		}
		if score > second {
			second = score
		}
	})

	if best.selection == nil {
		return "", false
	}

	if best.score < 30 {
		return "", false
	}
	if second > 0 && best.score < second*1.35 {
		return "", false
	}

	htmlStr, err := best.selection.Html()
	if err != nil || strings.TrimSpace(htmlStr) == "" {
		return "", false
	}
	return htmlStr, true
}

func scorePrimaryCandidate(s *goquery.Selection) float64 {
	text := strings.TrimSpace(normalizeText(s.Text()))
	if text == "" {
		return 0
	}

	lower := strings.ToLower(text)
	anchorCount := s.Find("a[href]").Length()
	structureCount := s.Find("p,h1,h2,h3,h4,h5,h6,ul,ol").Length()
	score := float64(len([]rune(text)))
	score += float64(s.Find("p").Length() * 18)
	score += float64(s.Find("h1,h2,h3,h4,h5,h6").Length() * 24)
	score += float64(minInt(anchorCount, 2) * 2)
	score += float64(s.Find("img[alt]").Length() * 8)
	score += float64(s.Find("table").Length() * 6)

	for _, phrase := range []string{
		"unsubscribe",
		"manage preferences",
		"view in browser",
		"read in browser",
		"view online",
		"preview text",
	} {
		if strings.Contains(lower, phrase) {
			score -= 40
		}
	}

	if structureCount == 0 {
		score *= 0.35
	}
	if anchorCount >= 3 && structureCount <= 1 {
		score *= 0.35
	}

	return math.Max(score, 0)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isLikelyNoiseLink(s *goquery.Selection) bool {
	text := strings.ToLower(strings.TrimSpace(normalizeText(s.Text())))
	href := strings.ToLower(strings.TrimSpace(getAttr(s, "href")))

	for _, phrase := range []string{
		"unsubscribe",
		"manage preferences",
		"update preferences",
		"email preferences",
		"view in browser",
		"read in browser",
		"view online",
		"opt out",
	} {
		if strings.Contains(text, phrase) || strings.Contains(href, phrase) {
			return true
		}
	}

	return false
}

func closestNoiseContainer(s *goquery.Selection) *goquery.Selection {
	for current := s; current != nil && current.Length() > 0; current = current.Parent() {
		if current.Is("p,div,td,li") && strings.TrimSpace(normalizeText(current.Text())) == strings.TrimSpace(normalizeText(s.Text())) {
			return current
		}
		if current.Is("body,html") {
			break
		}
	}
	return nil
}

func getAttr(s *goquery.Selection, name string) string {
	if s == nil {
		return ""
	}
	value, _ := s.Attr(name)
	return value
}

func filterAttrs(node *html.Node) {
	if node == nil {
		return
	}

	attrs := node.Attr[:0]
	for _, attr := range node.Attr {
		switch attr.Key {
		case "href", "src", "alt", "title":
			if attr.Key == "href" {
				attr.Val = cleanURL(attr.Val)
			}
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
