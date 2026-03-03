package livejournal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLToMarkdown_Basic(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "plain text",
			html:     "Hello world",
			expected: "Hello world",
		},
		{
			name:     "paragraph",
			html:     "<p>Hello world</p>",
			expected: "Hello world",
		},
		{
			name:     "bold with strong",
			html:     "<strong>bold text</strong>",
			expected: "**bold text**",
		},
		{
			name:     "bold with b",
			html:     "<b>bold text</b>",
			expected: "**bold text**",
		},
		{
			name:     "italic with em",
			html:     "<em>italic text</em>",
			expected: "*italic text*",
		},
		{
			name:     "italic with i",
			html:     "<i>italic text</i>",
			expected: "*italic text*",
		},
		{
			name:     "inline code",
			html:     "<code>code</code>",
			expected: "`code`",
		},
		{
			name:     "link",
			html:     `<a href="https://example.com">click here</a>`,
			expected: "[click here](https://example.com)",
		},
		{
			name:     "link without text",
			html:     `<a href="https://example.com"></a>`,
			expected: "[https://example.com](https://example.com)",
		},
		{
			name:     "image",
			html:     `<img src="https://example.com/img.jpg" alt="my image">`,
			expected: "![my image](https://example.com/img.jpg)",
		},
		{
			name:     "image without alt",
			html:     `<img src="https://example.com/img.jpg">`,
			expected: "![image](https://example.com/img.jpg)",
		},
		{
			name:     "line break",
			html:     "line1<br>line2",
			expected: "line1\nline2",
		},
		{
			name:     "horizontal rule",
			html:     "<hr>",
			expected: "---",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdown_Headings(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "h1",
			html:     "<h1>Heading 1</h1>",
			expected: "# Heading 1",
		},
		{
			name:     "h2",
			html:     "<h2>Heading 2</h2>",
			expected: "## Heading 2",
		},
		{
			name:     "h3",
			html:     "<h3>Heading 3</h3>",
			expected: "### Heading 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdown_Lists(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name: "unordered list",
			html: `<ul>
<li>Item 1</li>
<li>Item 2</li>
<li>Item 3</li>
</ul>`,
			expected: "- Item 1\n- Item 2\n- Item 3",
		},
		{
			name: "ordered list",
			html: `<ol>
<li>First</li>
<li>Second</li>
<li>Third</li>
</ol>`,
			expected: "1. First\n2. Second\n3. Third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdown_Blockquote(t *testing.T) {
	html := "<blockquote>This is a quote</blockquote>"
	result := HTMLToMarkdownWithLinkResolver(html, nil)
	assert.Equal(t, "> This is a quote", result)
}

func TestHTMLToMarkdown_Complex(t *testing.T) {
	html := `<p>This is a <strong>bold</strong> and <em>italic</em> paragraph.</p>
<ul>
<li>List item 1</li>
<li>List item 2</li>
</ul>
<blockquote>A quoted text</blockquote>
<p>Check out <a href="https://example.com">this link</a> for more.</p>`

	result := HTMLToMarkdownWithLinkResolver(html, nil)

	// Verify key conversions happened
	assert.Contains(t, result, "**bold**")
	assert.Contains(t, result, "*italic*")
	assert.Contains(t, result, "- List item 1")
	assert.Contains(t, result, "- List item 2")
	assert.Contains(t, result, "> A quoted text")
	assert.Contains(t, result, "[this link](https://example.com)")

	// Verify no HTML tags remain
	assert.NotContains(t, result, "<strong>")
	assert.NotContains(t, result, "<em>")
	assert.NotContains(t, result, "<li>")
	assert.NotContains(t, result, "<blockquote>")
	assert.NotContains(t, result, "<a ")
}

func TestHTMLToMarkdown_PreservesPlainMarkdown(t *testing.T) {
	// If input is already markdown (no HTML), it should pass through unchanged
	markdown := "This is **bold** and *italic* text.\n\n- Item 1\n- Item 2"
	result := HTMLToMarkdownWithLinkResolver(markdown, nil)
	assert.Equal(t, markdown, result)
}

func TestHTMLToMarkdown_RoundTrip(t *testing.T) {
	// Test that markdown -> HTML -> markdown preserves semantic content
	// This simulates: user writes markdown, it gets converted to HTML by LJ,
	// then we fetch it back and convert to markdown

	// Simulated HTML that LJ might store (simplified)
	// In reality, LJ's HTML might be slightly different
	htmlFromLJ := `<p>This is <strong>bold</strong> and <em>italic</em> text.</p>

<ul>
<li>List item 1</li>
<li>List item 2</li>
</ul>

<blockquote>A blockquote</blockquote>

<p><a href="https://example.com">A link</a></p>`

	// Convert HTML back to markdown
	convertedMarkdown := HTMLToMarkdownWithLinkResolver(htmlFromLJ, nil)

	// The converted markdown should contain the same semantic elements
	// (exact formatting may differ slightly)
	assert.Contains(t, convertedMarkdown, "**bold**")
	assert.Contains(t, convertedMarkdown, "*italic*")
	assert.Contains(t, convertedMarkdown, "- List item 1")
	assert.Contains(t, convertedMarkdown, "- List item 2")
	assert.Contains(t, convertedMarkdown, "> A blockquote")
	assert.Contains(t, convertedMarkdown, "[A link](https://example.com)")

	// Verify no HTML remains
	assert.NotContains(t, convertedMarkdown, "<")
	assert.NotContains(t, convertedMarkdown, ">A link<") // Make sure it's not raw HTML
}

func TestHTMLToMarkdown_LJEmbed(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "youtube embed to URL",
			html:     `<lj-embed source="youtube" vid="dQw4w9WgXcQ"></lj-embed>`,
			expected: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:     "vimeo embed to URL",
			html:     `<lj-embed source="vimeo" vid="123456"></lj-embed>`,
			expected: "https://vimeo.com/123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdown_UserLink(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "livejournal user link",
			html:     `<a href="https://john_doe.livejournal.com/">@john_doe</a>`,
			expected: "@john_doe",
		},
		{
			name:     "dreamwidth user link",
			html:     `<a href="https://alice.dreamwidth.org/">@alice</a>`,
			expected: "@alice",
		},
		{
			name:     "lj user tag",
			html:     `<lj user="bob">`,
			expected: "@bob",
		},
		{
			name:     "user link with different text preserved",
			html:     `<a href="https://john_doe.livejournal.com/">John's Journal</a>`,
			expected: "[John's Journal](https://john_doe.livejournal.com/)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdown_RoundTripYouTube(t *testing.T) {
	// Test that YouTube URL -> HTML embed -> YouTube URL works
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	// Start with a YouTube URL
	originalMD := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

	// Convert to HTML (which creates lj-embed)
	html := MarkdownToHTML(originalMD, config)
	assert.Contains(t, html, `<lj-embed source="youtube" vid="dQw4w9WgXcQ">`)

	// Convert back to markdown
	resultMD := HTMLToMarkdownWithLinkResolver(html, nil)
	assert.Equal(t, originalMD, resultMD)
}

func TestHTMLToMarkdown_RoundTripUserHandle(t *testing.T) {
	// Test that @username -> HTML link -> @username works
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	// Start with a user handle
	originalMD := "Hello @john_doe"

	// Convert to HTML
	html := MarkdownToHTML(originalMD, config)
	assert.Contains(t, html, `<a href="https://john_doe.livejournal.com/">@john_doe</a>`)

	// Convert back to markdown
	resultMD := HTMLToMarkdownWithLinkResolver(html, nil)
	assert.Contains(t, resultMD, "@john_doe")
}

func TestHTMLToMarkdown_RoundTripComplex(t *testing.T) {
	// Test a complex document with multiple features
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	originalMD := `Hey @alice, check out this video:

https://www.youtube.com/watch?v=dQw4w9WgXcQ

What do you think?`

	// Convert to HTML
	html := MarkdownToHTML(originalMD, config)

	// Convert back to markdown
	resultMD := HTMLToMarkdownWithLinkResolver(html, nil)

	// Should preserve the key elements
	assert.Contains(t, resultMD, "@alice")
	assert.Contains(t, resultMD, "https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	assert.Contains(t, resultMD, "What do you think?")
}
