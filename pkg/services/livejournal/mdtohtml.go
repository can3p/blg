package livejournal

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// YouTube URL patterns
var (
	youtubeRE = regexp.MustCompile(`^https?://(?:www\.)?(?:youtube(?:-nocookie)?\.com/(?:[^/]+/.+/|(?:v|e(?:mbed)?)/|.*[?&]v=)|youtu\.be/)([^"&?/\s]{11})(?:[?&][^\s]*)?$`)
	// Pattern for YouTube links within text (not just standalone)
	// Use non-greedy matching and explicit character classes to avoid over-matching
	youtubeLinkRE = regexp.MustCompile(`https?://(?:www\.)?(?:youtube(?:-nocookie)?\.com/(?:watch\?v=|embed/|v/)|youtu\.be/)([a-zA-Z0-9_-]{11})(?:[?&][^\s]*)?`)
	// @username pattern - must be preceded by whitespace or start of string
	usernameRE = regexp.MustCompile(`^@([a-z][0-9a-z]*(?:_[0-9a-z]+)*)`)
)

// ServiceConfig holds service-specific configuration for markdown conversion
type ServiceConfig struct {
	ServiceHost string // e.g., "livejournal.com" or "dreamwidth.org"
}

// YouTubeEmbedCode generates the LJ-style embed code for a YouTube video
func YouTubeEmbedCode(videoID string) string {
	return fmt.Sprintf(`<lj-embed source="youtube" vid="%s"></lj-embed>`, videoID)
}

// UserLink generates a link to a user's journal
func UserLink(username, serviceHost string) string {
	return fmt.Sprintf(`<a href="https://%s.%s/">@%s</a>`, username, serviceHost, username)
}

// LJUserTag generates the LJ-specific user tag
func LJUserTag(username string) string {
	return fmt.Sprintf(`<lj user="%s">`, username)
}

// MarkdownToHTML converts markdown to HTML with LiveJournal-specific extensions:
// - YouTube links in standalone paragraphs become embeds
// - YouTube links in text get an embed appended after the paragraph
// - @username references become user links
func MarkdownToHTML(md string, config ServiceConfig) string {
	// Create a custom goldmark instance with our extensions
	gm := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithInlineParsers(
				util.Prioritized(&userHandleParser{}, 999),
			),
			parser.WithASTTransformers(
				util.Prioritized(&youtubeTransformer{}, 100),
			),
		),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(&userHandleRenderer{config: config}, 500),
				util.Prioritized(&youtubeEmbedRenderer{}, 500),
			),
		),
	)

	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return md
	}

	return strings.TrimSpace(buf.String())
}

// --- YouTube Embed AST Node ---

var KindYouTubeEmbed = ast.NewNodeKind("YouTubeEmbed")

type YouTubeEmbed struct {
	ast.BaseBlock
	VideoID string
}

func (n *YouTubeEmbed) Kind() ast.NodeKind {
	return KindYouTubeEmbed
}

func (n *YouTubeEmbed) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"VideoID": n.VideoID}, nil)
}

// --- YouTube Transformer ---

type youtubeTransformer struct{}

func (t *youtubeTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()

	var nodesToProcess []*ast.Paragraph
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if p, ok := n.(*ast.Paragraph); ok {
			nodesToProcess = append(nodesToProcess, p)
		}
		return ast.WalkContinue, nil
	})

	for _, p := range nodesToProcess {
		t.processParagraph(p, source)
	}
}

func (t *youtubeTransformer) processParagraph(p *ast.Paragraph, source []byte) {
	// Get paragraph text by collecting all text segments
	var textContent strings.Builder
	_ = ast.Walk(p, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if textNode, ok := n.(*ast.Text); ok {
			textContent.Write(textNode.Segment.Value(source))
		} else if autoLink, ok := n.(*ast.AutoLink); ok {
			textContent.Write(autoLink.URL(source))
		}
		return ast.WalkContinue, nil
	})

	text := strings.TrimSpace(textContent.String())

	// Check if the entire paragraph is just a YouTube URL
	if match := youtubeRE.FindStringSubmatch(text); match != nil {
		// Replace paragraph with YouTube embed
		embed := &YouTubeEmbed{VideoID: match[1]}
		embed.SetBlankPreviousLines(p.HasBlankPreviousLines())
		p.Parent().ReplaceChild(p.Parent(), p, embed)
		return
	}

	// Check if paragraph contains YouTube links (but also other content)
	matches := youtubeLinkRE.FindAllStringSubmatch(text, -1)
	if len(matches) > 0 {
		// Add embeds after the paragraph for each YouTube link
		parent := p.Parent()
		var insertAfter ast.Node = p

		// Deduplicate video IDs
		seen := make(map[string]bool)
		for _, match := range matches {
			videoID := match[1]
			if seen[videoID] {
				continue
			}
			seen[videoID] = true

			embed := &YouTubeEmbed{VideoID: videoID}
			parent.InsertAfter(parent, insertAfter, embed)
			insertAfter = embed
		}
	}
}

// --- YouTube Embed Renderer ---

type youtubeEmbedRenderer struct {
	html.Config
}

func (r *youtubeEmbedRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindYouTubeEmbed, r.renderYouTubeEmbed)
}

func (r *youtubeEmbedRenderer) renderYouTubeEmbed(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*YouTubeEmbed)
	_, _ = w.WriteString(YouTubeEmbedCode(n.VideoID))
	_ = w.WriteByte('\n')

	return ast.WalkSkipChildren, nil
}

// --- User Handle AST Node ---

var KindUserHandle = ast.NewNodeKind("UserHandle")

type UserHandle struct {
	ast.BaseInline
	Username string
}

func (n *UserHandle) Kind() ast.NodeKind {
	return KindUserHandle
}

func (n *UserHandle) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"Username": n.Username}, nil)
}

// --- User Handle Parser ---

type userHandleParser struct{}

func (p *userHandleParser) Trigger() []byte {
	return []byte{'@'}
}

func (p *userHandleParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	if pc.IsInLinkLabel() {
		return nil
	}

	line, segment := block.PeekLine()
	if len(line) == 0 || line[0] != '@' {
		return nil
	}

	// Check if preceded by non-whitespace (e.g., email address)
	if segment.Start > 0 {
		source := block.Source()
		if segment.Start > 0 {
			prevChar := source[segment.Start-1]
			// If previous char is not whitespace or punctuation that typically precedes @mentions
			if prevChar != ' ' && prevChar != '\t' && prevChar != '\n' && prevChar != '(' && prevChar != '[' && prevChar != ',' {
				return nil
			}
		}
	}

	// Match @username pattern
	match := usernameRE.FindSubmatch(line)
	if len(match) < 2 {
		return nil
	}

	username := string(match[1])
	if len(username) < 3 {
		return nil
	}

	// Advance past the matched text
	block.Advance(len(match[0]))

	return &UserHandle{Username: username}
}

// --- User Handle Renderer ---

type userHandleRenderer struct {
	html.Config
	config ServiceConfig
}

func (r *userHandleRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindUserHandle, r.renderUserHandle)
}

func (r *userHandleRenderer) renderUserHandle(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*UserHandle)

	if r.config.ServiceHost != "" {
		_, _ = w.WriteString(UserLink(n.Username, r.config.ServiceHost))
	} else {
		// Fallback to LJ user tag if no service host configured
		_, _ = w.WriteString(LJUserTag(n.Username))
	}

	return ast.WalkSkipChildren, nil
}
