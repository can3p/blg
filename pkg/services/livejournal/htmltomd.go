package livejournal

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Regex to extract username from LJ-style user URLs
var userLinkRE = regexp.MustCompile(`^https?://([a-z][0-9a-z_]+)\.(?:livejournal\.com|dreamwidth\.org)/?$`)

// HTMLToMarkdown converts HTML content to markdown format.
// This is used when fetching posts from LiveJournal that were created
// via the web UI (which stores content as HTML).
// See cl-journal/src/markdownify.lisp for reference implementation.
func HTMLToMarkdown(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	var sb strings.Builder
	convertNode(&sb, doc)
	return strings.TrimSpace(sb.String())
}

func convertNode(sb *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		text := n.Data
		// Collapse multiple whitespace but preserve single spaces
		if strings.TrimSpace(text) != "" {
			sb.WriteString(text)
		} else if text != "" && (strings.Contains(text, " ") || strings.Contains(text, "\n")) {
			// Preserve a single space for whitespace-only text nodes between elements
			sb.WriteString(" ")
		}
	case html.ElementNode:
		convertElement(sb, n)
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
	}
}

func convertElement(sb *strings.Builder, n *html.Node) {
	tag := strings.ToLower(n.Data)

	switch tag {
	case "html", "head", "body":
		// Skip wrapper elements, just process children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}

	case "p":
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("\n\n")

	case "br":
		sb.WriteString("\n")

	case "strong", "b":
		sb.WriteString("**")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("**")

	case "em", "i":
		sb.WriteString("*")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("*")

	case "code":
		sb.WriteString("`")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("`")

	case "pre":
		sb.WriteString("```\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("\n```\n\n")

	case "a":
		href := getAttr(n, "href")
		var linkText strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(&linkText, c)
		}
		text := strings.TrimSpace(linkText.String())
		if text == "" {
			text = href
		}

		// Check if this is a user link (e.g., https://username.livejournal.com/)
		if match := userLinkRE.FindStringSubmatch(href); match != nil {
			username := match[1]
			// If link text is @username, just output @username
			if text == "@"+username {
				sb.WriteString("@")
				sb.WriteString(username)
				return
			}
		}

		sb.WriteString("[")
		sb.WriteString(text)
		sb.WriteString("](")
		sb.WriteString(href)
		sb.WriteString(")")

	case "img":
		src := getAttr(n, "src")
		alt := getAttr(n, "alt")
		if alt == "" {
			alt = "image"
		}
		sb.WriteString("![")
		sb.WriteString(alt)
		sb.WriteString("](")
		sb.WriteString(src)
		sb.WriteString(")")

	case "ul":
		sb.WriteString("\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && strings.ToLower(c.Data) == "li" {
				sb.WriteString("- ")
				for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
					convertNode(sb, cc)
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")

	case "ol":
		sb.WriteString("\n")
		num := 1
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && strings.ToLower(c.Data) == "li" {
				sb.WriteString(strings.Repeat(" ", 0))
				sb.WriteString(string(rune('0'+num)) + ". ")
				for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
					convertNode(sb, cc)
				}
				sb.WriteString("\n")
				num++
			}
		}
		sb.WriteString("\n")

	case "blockquote":
		var quoteContent strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(&quoteContent, c)
		}
		lines := strings.Split(strings.TrimSpace(quoteContent.String()), "\n")
		for _, line := range lines {
			sb.WriteString("> ")
			sb.WriteString(strings.TrimSpace(line))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")

	case "h1":
		sb.WriteString("\n# ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("\n\n")

	case "h2":
		sb.WriteString("\n## ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("\n\n")

	case "h3":
		sb.WriteString("\n### ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("\n\n")

	case "h4":
		sb.WriteString("\n#### ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("\n\n")

	case "h5":
		sb.WriteString("\n##### ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("\n\n")

	case "h6":
		sb.WriteString("\n###### ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
		sb.WriteString("\n\n")

	case "hr":
		sb.WriteString("\n---\n\n")

	case "div", "span":
		// Just process children for generic containers
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}

	case "lj-embed":
		// Handle LJ-specific embeds (YouTube, Vimeo) - convert back to URLs
		source := getAttr(n, "source")
		vid := getAttr(n, "vid")
		switch source {
		case "youtube":
			// Convert back to YouTube URL
			sb.WriteString(fmt.Sprintf("https://www.youtube.com/watch?v=%s", vid))
		case "vimeo":
			// Convert back to Vimeo URL
			sb.WriteString(fmt.Sprintf("https://vimeo.com/%s", vid))
		default:
			// Unknown embed, keep as-is
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				convertNode(sb, c)
			}
		}

	case "lj":
		// Handle <lj user="username"> tag - convert to @username
		user := getAttr(n, "user")
		if user != "" {
			sb.WriteString("@")
			sb.WriteString(user)
		}

	default:
		// For unknown elements, just process children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNode(sb, c)
		}
	}
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}
