package parser

import (
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/nonozone/MailCli/pkg/schema"
)

var urlRegex = regexp.MustCompile(`https?://[^\s>"]+`)

func extractActions(meta schema.MessageMeta, htmlBody string, abuseTargets ...string) []schema.Action {
	seen := map[string]struct{}{}
	var actions []schema.Action

	for _, value := range meta.ListUnsubscribe {
		if target := normalizeHeaderLinkTarget(value); target != "" {
			appendAction(&actions, seen, schema.Action{
				Type:  "unsubscribe",
				Label: "Unsubscribe",
				URL:   normalizeActionURL(target),
			})
			continue
		}
		for _, url := range extractURLs(value) {
			appendAction(&actions, seen, schema.Action{
				Type:  "unsubscribe",
				Label: "Unsubscribe",
				URL:   normalizeActionURL(url),
			})
		}
	}

	for _, target := range abuseTargets {
		target = normalizeAbuseTarget(target)
		appendAction(&actions, seen, schema.Action{
			Type:  "report_abuse",
			Label: "Report abuse",
			URL:   target,
		})
	}

	for _, action := range extractAnchorActions(htmlBody) {
		appendAction(&actions, seen, action)
	}

	return actions
}

func splitHeaderLinks(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	matches := regexp.MustCompile(`<([^>]+)>`).FindAllStringSubmatch(value, -1)
	if len(matches) > 0 {
		out := make([]string, 0, len(matches))
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			if target := normalizeHeaderLinkTarget(match[1]); target != "" {
				out = append(out, target)
			}
		}
		if len(out) > 0 {
			return out
		}
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if target := normalizeHeaderLinkTarget(part); target != "" {
			out = append(out, target)
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

func normalizeHeaderLinkTarget(value string) string {
	trimmed := strings.TrimSpace(strings.Trim(value, "<>"))
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "mailto:"):
		return trimmed
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return trimmed
	case strings.Contains(trimmed, "@") && !strings.Contains(trimmed, " "):
		return "mailto:" + trimmed
	default:
		return ""
	}
}

func normalizeActionURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(trimmed), "mailto:") {
		return trimmed
	}
	return cleanURL(trimmed)
}

func cleanURL(value string) string {
	return cleanURLDepth(value, 0)
}

func cleanURLDepth(value string, depth int) string {
	if depth >= 4 {
		return value
	}

	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil {
		return value
	}
	if !looksLikeRedirectWrapper(parsed) {
		return value
	}

	if target := extractWrappedTarget(parsed); target != "" {
		if cleaned := cleanURLDepth(target, depth+1); cleaned != "" {
			return cleaned
		}
		return target
	}

	return value
}

func extractWrappedTarget(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}

	for _, key := range []string{"url", "u", "a", "target", "redirect", "redirect_url", "redirect_uri", "dest", "destination"} {
		if target := decodeURLCandidate(parsed.Query().Get(key)); target != "" {
			return target
		}
	}

	pathSegments := strings.Split(parsed.EscapedPath(), "/")
	for i := len(pathSegments) - 1; i >= 0; i-- {
		if target := decodeURLCandidate(pathSegments[i]); target != "" {
			return target
		}
	}

	return ""
}

func decodeURLCandidate(value string) string {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return ""
	}

	for range 3 {
		if target, ok := parseAbsoluteHTTPURL(candidate); ok {
			return target
		}

		decoded, err := url.QueryUnescape(candidate)
		if err != nil || decoded == candidate {
			break
		}
		candidate = strings.TrimSpace(decoded)
	}

	if target, ok := parseAbsoluteHTTPURL(candidate); ok {
		return target
	}
	return ""
}

func parseAbsoluteHTTPURL(value string) (string, bool) {
	target, err := url.Parse(strings.TrimSpace(value))
	if err != nil || target == nil {
		return "", false
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return "", false
	}
	if target.Host == "" {
		return "", false
	}
	return target.String(), true
}

func extractAnchorActions(htmlBody string) []schema.Action {
	if strings.TrimSpace(htmlBody) == "" {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil
	}

	var actions []schema.Action
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}

		label := normalizeActionLabel(s.Text())
		actionType := classifyAction(label, href)
		if actionType == "" {
			return
		}

		actions = append(actions, schema.Action{
			Type:  actionType,
			Label: actionLabel(actionType, label),
			URL:   cleanURL(strings.TrimSpace(href)),
		})
	})

	return actions
}

func classifyAction(label, href string) string {
	lowerLabel := strings.ToLower(label)
	lowerHref := strings.ToLower(strings.TrimSpace(href))

	switch {
	case strings.Contains(lowerLabel, "unsubscribe") || strings.Contains(lowerHref, "unsubscribe"):
		return "unsubscribe"
	case looksLikeResetPassword(lowerLabel, lowerHref):
		return "reset_password"
	case looksLikeVerifySignIn(lowerLabel, lowerHref):
		return "verify_sign_in"
	case looksLikeDownloadAttachment(lowerLabel, lowerHref):
		return "download_attachment"
	case looksLikeViewAttachment(lowerLabel, lowerHref):
		return "view_attachment"
	case looksLikePayInvoice(lowerLabel, lowerHref):
		return "pay_invoice"
	case looksLikeViewInvoice(lowerLabel, lowerHref):
		return "view_invoice"
	case strings.Contains(lowerLabel, "view online") || strings.Contains(lowerLabel, "open in browser") || strings.Contains(lowerLabel, "read in browser") || strings.Contains(lowerHref, "view-online"):
		return "view_online"
	case strings.Contains(lowerLabel, "confirm subscription") || strings.Contains(lowerLabel, "confirm email") || strings.Contains(lowerHref, "confirm-subscription") || strings.Contains(lowerHref, "confirm-email"):
		return "confirm_subscription"
	case strings.Contains(lowerLabel, "report abuse") || strings.HasPrefix(lowerHref, "mailto:abuse@") || strings.Contains(lowerHref, "report-abuse"):
		return "report_abuse"
	default:
		return ""
	}
}

func actionLabel(actionType, label string) string {
	if label != "" {
		return label
	}

	switch actionType {
	case "unsubscribe":
		return "Unsubscribe"
	case "reset_password":
		return "Reset password"
	case "verify_sign_in":
		return "Verify sign-in"
	case "pay_invoice":
		return "Pay invoice"
	case "view_invoice":
		return "View invoice"
	case "download_attachment":
		return "Download attachment"
	case "view_attachment":
		return "View attachment"
	case "view_online":
		return "View online"
	case "confirm_subscription":
		return "Confirm subscription"
	case "report_abuse":
		return "Report abuse"
	default:
		return ""
	}
}

func normalizeActionLabel(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func appendAction(actions *[]schema.Action, seen map[string]struct{}, action schema.Action) {
	if strings.TrimSpace(action.Type) == "" || strings.TrimSpace(action.URL) == "" {
		return
	}

	key := action.Type + "\x00" + action.URL
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*actions = append(*actions, action)
}

func normalizeAbuseTarget(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "mailto:") || strings.HasPrefix(strings.ToLower(trimmed), "http://") || strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		return trimmed
	}
	if strings.Contains(trimmed, "@") && !strings.Contains(trimmed, " ") {
		return "mailto:" + trimmed
	}
	return ""
}

func looksLikeRedirectWrapper(parsed *url.URL) bool {
	if parsed == nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	fullPath := strings.ToLower(path.Clean(parsed.Path))
	combined := host + " " + fullPath

	for _, token := range []string{"proofpoint", "urldefense", "safelinks.protection.outlook.com"} {
		if strings.Contains(host, token) {
			return true
		}
	}

	for _, token := range []string{"click", "track", "redirect", "redir", "out", "away", "link", "lnk"} {
		if strings.Contains(combined, token) {
			return true
		}
	}

	return false
}

func looksLikeDownloadAttachment(label, href string) bool {
	if strings.Contains(label, "download attachment") || strings.Contains(label, "download file") || strings.Contains(label, "download invoice") {
		return true
	}

	if strings.Contains(href, "/download/") && strings.Contains(href, "attachment") {
		return true
	}

	return strings.Contains(href, "download=1") && strings.Contains(href, "attachment")
}

func looksLikeViewAttachment(label, href string) bool {
	if strings.Contains(label, "view attachment") || strings.Contains(label, "open attachment") {
		return true
	}

	if strings.Contains(href, "attachment") && !strings.Contains(href, "/download/") && !strings.Contains(href, "download=1") {
		return true
	}

	return false
}

func looksLikeResetPassword(label, href string) bool {
	if strings.Contains(label, "reset password") || strings.Contains(label, "change password") {
		return true
	}

	return strings.Contains(href, "/reset-password") || strings.Contains(href, "/password-reset")
}

func looksLikeVerifySignIn(label, href string) bool {
	if strings.Contains(label, "verify sign-in") || strings.Contains(label, "verify signin") || strings.Contains(label, "approve sign-in") || strings.Contains(label, "approve login") {
		return true
	}

	return strings.Contains(href, "/verify-sign-in") || strings.Contains(href, "/verify-login") || strings.Contains(href, "/approve-login")
}

func looksLikePayInvoice(label, href string) bool {
	if strings.Contains(label, "pay invoice") || strings.Contains(label, "pay bill") {
		return true
	}

	return hasInvoicePath(href) && strings.Contains(href, "/pay")
}

func looksLikeViewInvoice(label, href string) bool {
	if strings.Contains(label, "view invoice") || strings.Contains(label, "open invoice") || strings.Contains(label, "see invoice") {
		return true
	}

	return false
}

func hasInvoicePath(href string) bool {
	return strings.Contains(href, "/invoice/") || strings.Contains(href, "/invoices/")
}
