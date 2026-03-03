package livejournal

import (
	"net/url"
	"strings"

	"github.com/can3p/blg/pkg/types"
	"golang.org/x/net/html"
)

// Re-export types for backward compatibility
type LinkResolver = types.LinkResolver
type PostURLMapping = types.PostURLMapping

// BuildLinkResolver creates a LinkResolver from a list of URL-to-filename mappings.
var BuildLinkResolver = types.BuildLinkResolver

// HTMLToMarkdownWithLinkResolver converts HTML content to markdown format,
// resolving internal post links to local filenames using the provided resolver.
// If resolver is nil, links are preserved as-is.
func HTMLToMarkdownWithLinkResolver(htmlContent string, resolver LinkResolver) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	var sb strings.Builder
	convertNodeWithResolver(&sb, doc, resolver)
	return strings.TrimSpace(sb.String())
}

func convertNodeWithResolver(sb *strings.Builder, n *html.Node, resolver LinkResolver) {
	switch n.Type {
	case html.TextNode:
		text := n.Data
		if strings.TrimSpace(text) != "" {
			sb.WriteString(text)
		} else if text != "" && (strings.Contains(text, " ") || strings.Contains(text, "\n")) {
			sb.WriteString(" ")
		}
	case html.ElementNode:
		convertElementWithResolver(sb, n, resolver)
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
	}
}

func convertElementWithResolver(sb *strings.Builder, n *html.Node, resolver LinkResolver) {
	tag := strings.ToLower(n.Data)

	switch tag {
	case "html", "head", "body":
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}

	case "p":
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("\n\n")

	case "br":
		sb.WriteString("\n")

	case "strong", "b":
		sb.WriteString("**")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("**")

	case "em", "i":
		sb.WriteString("*")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("*")

	case "code":
		sb.WriteString("`")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("`")

	case "pre":
		sb.WriteString("```\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("\n```\n\n")

	case "a":
		href := getAttr(n, "href")
		var linkText strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(&linkText, c, resolver)
		}
		text := strings.TrimSpace(linkText.String())

		// Check if this is a user link (e.g., https://username.livejournal.com/)
		if match := userLinkRE.FindStringSubmatch(href); match != nil {
			username := match[1]
			if text == "@"+username {
				sb.WriteString("@")
				sb.WriteString(username)
				return
			}
		}

		// Try to resolve the link to a local filename
		resolvedHref := href
		if resolver != nil {
			resolvedHref = resolveLink(href, resolver)
		}

		// If link text is empty, use the resolved filename or original URL
		if text == "" {
			text = resolvedHref
		}

		sb.WriteString("[")
		sb.WriteString(text)
		sb.WriteString("](")
		sb.WriteString(resolvedHref)
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
					convertNodeWithResolver(sb, cc, resolver)
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
				sb.WriteString(string(rune('0'+num)) + ". ")
				for cc := c.FirstChild; cc != nil; cc = cc.NextSibling {
					convertNodeWithResolver(sb, cc, resolver)
				}
				sb.WriteString("\n")
				num++
			}
		}
		sb.WriteString("\n")

	case "blockquote":
		var quoteContent strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(&quoteContent, c, resolver)
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
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("\n\n")

	case "h2":
		sb.WriteString("\n## ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("\n\n")

	case "h3":
		sb.WriteString("\n### ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("\n\n")

	case "h4":
		sb.WriteString("\n#### ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("\n\n")

	case "h5":
		sb.WriteString("\n##### ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("\n\n")

	case "h6":
		sb.WriteString("\n###### ")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
		sb.WriteString("\n\n")

	case "hr":
		sb.WriteString("\n---\n\n")

	case "div", "span":
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}

	case "lj-embed":
		source := getAttr(n, "source")
		vid := getAttr(n, "vid")
		switch source {
		case "youtube":
			sb.WriteString("https://www.youtube.com/watch?v=" + vid)
		case "vimeo":
			sb.WriteString("https://vimeo.com/" + vid)
		default:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				convertNodeWithResolver(sb, c, resolver)
			}
		}

	case "lj":
		user := getAttr(n, "user")
		if user != "" {
			sb.WriteString("@")
			sb.WriteString(user)
		}

	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			convertNodeWithResolver(sb, c, resolver)
		}
	}
}

// resolveLink attempts to resolve a URL to a local filename.
// It handles URLs with anchors by preserving the anchor in the result.
func resolveLink(href string, resolver LinkResolver) string {
	// Parse the URL to extract base and fragment
	parsed, err := url.Parse(href)
	if err != nil {
		return href
	}

	// Build the base URL without fragment
	baseURL := *parsed
	baseURL.Fragment = ""
	baseURLStr := baseURL.String()

	// Try to resolve the base URL
	if filename, ok := resolver(baseURLStr); ok {
		// If there's a fragment, append it to the filename
		if parsed.Fragment != "" {
			return filename + "#" + parsed.Fragment
		}
		return filename
	}

	// Not found, return original href
	return href
}
