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
