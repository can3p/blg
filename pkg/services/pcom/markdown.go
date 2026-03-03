package pcom

import (
	"bytes"
	"strings"

	markdown "github.com/teekennedy/goldmark-markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type mdParser struct {
	source []byte
	parsed ast.Node
	parser goldmark.Markdown
}

func parseBody(s string) (*mdParser, error) {
	gm := goldmark.New(
		goldmark.WithRenderer(markdown.NewRenderer()),
	)

	r := text.NewReader([]byte(s))

	node := gm.Parser().Parse(r)

	return &mdParser{
		source: []byte(s),
		parsed: node,
		parser: gm,
	}, nil
}

func (p *mdParser) MaybeString() (string, error) {
	var b bytes.Buffer

	err := p.parser.Renderer().Render(&b, p.source, p.parsed)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(b.String()), nil
}

func (p *mdParser) ExtractImages() ([]string, error) {
	out := []string{}

	// Walk the AST in depth-first fashion and apply transformations
	err := ast.Walk(p.parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		// Each node will be visited twice, once when it is first encountered (entering), and again
		// after all the node's children have been visited (if any). Skip the latter.
		if !entering {
			return ast.WalkContinue, nil
		}

		if node.Kind() == ast.KindImage {
			imgNode := node.(*ast.Image)

			out = append(out, string(imgNode.Destination))
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}

func (p *mdParser) ReplaceImages(m map[string]string) error {
	// Walk the AST in depth-first fashion and apply transformations
	return ast.Walk(p.parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		// Each node will be visited twice, once when it is first encountered (entering), and again
		// after all the node's children have been visited (if any). Skip the latter.
		if !entering {
			return ast.WalkContinue, nil
		}

		if node.Kind() == ast.KindImage {
			imgNode := node.(*ast.Image)

			newValue, shouldReplace := m[string(imgNode.Destination)]

			if shouldReplace {
				imgNode.Destination = []byte(newValue)
			}
		}

		return ast.WalkContinue, nil
	})
}

func (p *mdParser) ExtractLinks() ([]string, error) {
	out := []string{}

	err := ast.Walk(p.parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if node.Kind() == ast.KindLink {
			linkNode := node.(*ast.Link)
			dest := string(linkNode.Destination)
			// Only extract local .md file links
			if strings.HasSuffix(dest, ".md") && !strings.Contains(dest, "://") {
				out = append(out, dest)
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, err
	}

	return out, nil
}

func (p *mdParser) ReplaceLinks(m map[string]string) error {
	return ast.Walk(p.parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if node.Kind() == ast.KindLink {
			linkNode := node.(*ast.Link)

			newValue, shouldReplace := m[string(linkNode.Destination)]

			if shouldReplace {
				linkNode.Destination = []byte(newValue)
			}
		}

		return ast.WalkContinue, nil
	})
}

// ReplaceLinksWithResolver replaces link destinations using a LinkResolver function.
// This allows resolving remote URLs to local filenames during sync.
// Links with anchors (e.g., url#section) are handled by stripping the anchor
// for lookup and re-appending it to the resolved filename.
func (p *mdParser) ReplaceLinksWithResolver(resolver func(string) (string, bool)) error {
	return ast.Walk(p.parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if node.Kind() == ast.KindLink {
			linkNode := node.(*ast.Link)
			dest := string(linkNode.Destination)

			// Try to resolve the link
			resolved := resolveURLWithAnchor(dest, resolver)
			if resolved != dest {
				linkNode.Destination = []byte(resolved)
			}
		}

		return ast.WalkContinue, nil
	})
}

// resolveURLWithAnchor attempts to resolve a URL to a local filename,
// preserving any anchor/fragment in the URL.
func resolveURLWithAnchor(href string, resolver func(string) (string, bool)) string {
	if resolver == nil {
		return href
	}

	// Split URL and anchor
	baseURL := href
	anchor := ""
	if idx := strings.Index(href, "#"); idx != -1 {
		baseURL = href[:idx]
		anchor = href[idx:]
	}

	// Try to resolve the base URL
	if filename, ok := resolver(baseURL); ok {
		return filename + anchor
	}

	return href
}
