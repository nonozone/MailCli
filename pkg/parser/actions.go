package parser

import (
	"net/url"
	"path"
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
			url = cleanURL(url)
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
		url = cleanURL(url)
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

func cleanURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil {
		return value
	}
	if !looksLikeRedirectWrapper(parsed) {
		return value
	}

	for _, key := range []string{"url", "target", "redirect", "redirect_url", "redirect_uri", "dest", "destination"} {
		candidate := strings.TrimSpace(parsed.Query().Get(key))
		if candidate == "" {
			continue
		}

		target, err := url.Parse(candidate)
		if err != nil || target == nil {
			continue
		}
		if target.Scheme != "http" && target.Scheme != "https" {
			continue
		}
		if target.Host == "" {
			continue
		}
		return target.String()
	}

	return value
}

func looksLikeRedirectWrapper(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	fullPath := strings.ToLower(path.Clean(parsed.Path))
	combined := host + " " + fullPath

	for _, token := range []string{"click", "track", "redirect", "redir", "out", "away", "link", "lnk"} {
		if strings.Contains(combined, token) {
			return true
		}
	}

	return false
}
