package parser

import (
	"regexp"
	"strings"

	"github.com/yourname/mailcli/pkg/schema"
)

var urlRegex = regexp.MustCompile(`https?://[^\s>"]+`)

func extractActions(meta schema.MessageMeta, htmlBody string) []schema.Action {
	seen := map[string]struct{}{}
	var actions []schema.Action

	for _, value := range meta.ListUnsubscribe {
		for _, url := range extractURLs(value) {
			if _, ok := seen[url]; ok {
				continue
			}
			seen[url] = struct{}{}
			actions = append(actions, schema.Action{
				Type:  "unsubscribe",
				Label: "Unsubscribe",
				URL:   url,
			})
		}
	}

	for _, url := range extractURLs(htmlBody) {
		if _, ok := seen[url]; ok {
			continue
		}
		if !strings.Contains(strings.ToLower(url), "unsubscribe") {
			continue
		}
		seen[url] = struct{}{}
		actions = append(actions, schema.Action{
			Type:  "unsubscribe",
			Label: "Unsubscribe",
			URL:   url,
		})
	}

	return actions
}

func splitHeaderLinks(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, "<>"))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func extractURLs(value string) []string {
	matches := urlRegex.FindAllString(value, -1)
	if len(matches) == 0 {
		return nil
	}
	return matches
}
