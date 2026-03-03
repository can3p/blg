package livejournal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLToMarkdownWithLinkResolver(t *testing.T) {
	// URL to filename mapping - simulates what we'd have after fetching posts
	urlToFilename := map[string]string{
		"https://can3p_test.livejournal.com/122.html": "that.md",
		"https://can3p_test.livejournal.com/123.html": "this.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "basic link resolution",
			html:     `<p>check the post <a href="https://can3p_test.livejournal.com/123.html">this one</a></p>`,
			expected: "check the post [this one](this.md)",
		},
		{
			name:     "reverse link resolution",
			html:     `<p>check the previous post <a href="https://can3p_test.livejournal.com/122.html">that one</a></p>`,
			expected: "check the previous post [that one](that.md)",
		},
		{
			name:     "multiple links in same post",
			html:     `<p>See <a href="https://can3p_test.livejournal.com/122.html">post A</a> and <a href="https://can3p_test.livejournal.com/123.html">post B</a></p>`,
			expected: "See [post A](that.md) and [post B](this.md)",
		},
		{
			name:     "external link preserved",
			html:     `<p>Check <a href="https://example.com/page">external site</a></p>`,
			expected: "Check [external site](https://example.com/page)",
		},
		{
			name:     "mixed internal and external links",
			html:     `<p>See <a href="https://can3p_test.livejournal.com/122.html">my post</a> and <a href="https://google.com">Google</a></p>`,
			expected: "See [my post](that.md) and [Google](https://google.com)",
		},
		{
			name:     "unknown internal link preserved",
			html:     `<p>See <a href="https://can3p_test.livejournal.com/999.html">unknown post</a></p>`,
			expected: "See [unknown post](https://can3p_test.livejournal.com/999.html)",
		},
		{
			name:     "link with no text uses filename",
			html:     `<p>See <a href="https://can3p_test.livejournal.com/122.html"></a></p>`,
			expected: "See [that.md](that.md)",
		},
		{
			name:     "self-referential link",
			html:     `<p>This is <a href="https://can3p_test.livejournal.com/122.html">the current post</a></p>`,
			expected: "This is [the current post](that.md)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, resolver)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdownWithLinkResolver_NilResolver(t *testing.T) {
	// When no resolver is provided, links should be preserved as-is
	html := `<p>check <a href="https://can3p_test.livejournal.com/123.html">this</a></p>`
	result := HTMLToMarkdownWithLinkResolver(html, nil)
	assert.Equal(t, "check [this](https://can3p_test.livejournal.com/123.html)", result)
}

func TestHTMLToMarkdownWithLinkResolver_ComplexDocument(t *testing.T) {
	urlToFilename := map[string]string{
		"https://user.livejournal.com/100.html": "intro.md",
		"https://user.livejournal.com/200.html": "chapter-1.md",
		"https://user.livejournal.com/300.html": "chapter-2.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	html := `<p>Welcome to my series!</p>
<ul>
<li><a href="https://user.livejournal.com/100.html">Introduction</a></li>
<li><a href="https://user.livejournal.com/200.html">Chapter 1</a></li>
<li><a href="https://user.livejournal.com/300.html">Chapter 2</a></li>
</ul>
<p>Also check <a href="https://example.com">my website</a>.</p>`

	result := HTMLToMarkdownWithLinkResolver(html, resolver)

	assert.Contains(t, result, "[Introduction](intro.md)")
	assert.Contains(t, result, "[Chapter 1](chapter-1.md)")
	assert.Contains(t, result, "[Chapter 2](chapter-2.md)")
	assert.Contains(t, result, "[my website](https://example.com)")
}

func TestHTMLToMarkdownWithLinkResolver_DreamwidthLinks(t *testing.T) {
	// Test that Dreamwidth links are also resolved
	urlToFilename := map[string]string{
		"https://user.dreamwidth.org/12345.html": "post.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	html := `<p>See <a href="https://user.dreamwidth.org/12345.html">my DW post</a></p>`
	result := HTMLToMarkdownWithLinkResolver(html, resolver)
	assert.Equal(t, "See [my DW post](post.md)", result)
}

func TestHTMLToMarkdownWithLinkResolver_UserLinksPreserved(t *testing.T) {
	// User profile links should still be converted to @username format
	resolver := func(url string) (string, bool) {
		return "", false
	}

	html := `<p>Thanks <a href="https://john_doe.livejournal.com/">@john_doe</a> for the help!</p>`
	result := HTMLToMarkdownWithLinkResolver(html, resolver)
	assert.Equal(t, "Thanks @john_doe for the help!", result)
}

func TestHTMLToMarkdownWithLinkResolver_AnchorLinks(t *testing.T) {
	// Links with anchors should be resolved to filename with anchor
	urlToFilename := map[string]string{
		"https://user.livejournal.com/100.html": "post.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "link with anchor",
			html:     `<p>See <a href="https://user.livejournal.com/100.html#section1">section 1</a></p>`,
			expected: "See [section 1](post.md#section1)",
		},
		{
			name:     "link with query params preserved for external",
			html:     `<p>See <a href="https://example.com/page?foo=bar">external</a></p>`,
			expected: "See [external](https://example.com/page?foo=bar)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, resolver)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdownWithLinkResolver_HTTPLinks(t *testing.T) {
	// Both http and https links should be resolved
	urlToFilename := map[string]string{
		"http://user.livejournal.com/100.html":  "post-http.md",
		"https://user.livejournal.com/100.html": "post-https.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "http link",
			html:     `<p>See <a href="http://user.livejournal.com/100.html">http post</a></p>`,
			expected: "See [http post](post-http.md)",
		},
		{
			name:     "https link",
			html:     `<p>See <a href="https://user.livejournal.com/100.html">https post</a></p>`,
			expected: "See [https post](post-https.md)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, resolver)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildLinkResolver(t *testing.T) {
	// Test the helper function that builds a resolver from post metadata
	posts := []PostURLMapping{
		{URL: "https://user.livejournal.com/100.html", Filename: "first-post.md"},
		{URL: "https://user.livejournal.com/200.html", Filename: "second-post.md"},
	}

	resolver := BuildLinkResolver(posts)

	fname, ok := resolver("https://user.livejournal.com/100.html")
	assert.True(t, ok)
	assert.Equal(t, "first-post.md", fname)

	fname, ok = resolver("https://user.livejournal.com/200.html")
	assert.True(t, ok)
	assert.Equal(t, "second-post.md", fname)

	_, ok = resolver("https://user.livejournal.com/999.html")
	assert.False(t, ok)
}

func TestHTMLToMarkdownWithLinkResolver_NestedLinks(t *testing.T) {
	// Links inside other elements should still be resolved
	urlToFilename := map[string]string{
		"https://user.livejournal.com/100.html": "post.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "link in bold",
			html:     `<p><strong><a href="https://user.livejournal.com/100.html">bold link</a></strong></p>`,
			expected: "**[bold link](post.md)**",
		},
		{
			name:     "link in italic",
			html:     `<p><em><a href="https://user.livejournal.com/100.html">italic link</a></em></p>`,
			expected: "*[italic link](post.md)*",
		},
		{
			name:     "link in list item",
			html:     `<ul><li>See <a href="https://user.livejournal.com/100.html">this post</a></li></ul>`,
			expected: "- See [this post](post.md)",
		},
		{
			name:     "link in blockquote",
			html:     `<blockquote>As mentioned in <a href="https://user.livejournal.com/100.html">this post</a></blockquote>`,
			expected: "> As mentioned in [this post](post.md)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, resolver)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdownWithLinkResolver_SpecialCharactersInURL(t *testing.T) {
	// URLs with special characters should be handled correctly
	urlToFilename := map[string]string{
		"https://user.livejournal.com/100.html": "post.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "URL with query string not in map",
			html:     `<p><a href="https://user.livejournal.com/100.html?mode=reply">reply</a></p>`,
			expected: "reply",
		},
		{
			name:     "URL with multiple fragments",
			html:     `<p><a href="https://user.livejournal.com/100.html#section#subsection">section</a></p>`,
			expected: "[section](post.md#section#subsection)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, resolver)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestHTMLToMarkdownWithLinkResolver_CommunityPosts(t *testing.T) {
	// Community posts have different URL patterns
	urlToFilename := map[string]string{
		"https://community.livejournal.com/100.html":          "community-post.md",
		"https://users.livejournal.com/user/100.html":         "user-post.md",
		"https://www.livejournal.com/users/user/100.html":     "www-user-post.md",
		"https://www.livejournal.com/community/comm/100.html": "www-comm-post.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "community subdomain",
			html:     `<p><a href="https://community.livejournal.com/100.html">comm post</a></p>`,
			expected: "[comm post](community-post.md)",
		},
		{
			name:     "users path",
			html:     `<p><a href="https://users.livejournal.com/user/100.html">user post</a></p>`,
			expected: "[user post](user-post.md)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, resolver)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdownWithLinkResolver_CircularReferences(t *testing.T) {
	// Two posts that reference each other - the main use case from the request
	urlToFilename := map[string]string{
		"https://can3p_test.livejournal.com/122.html": "that.md",
		"https://can3p_test.livejournal.com/123.html": "this.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	// Post 122 (that.md) references post 123 (this.md)
	html1 := `<p>check the post <a href="https://can3p_test.livejournal.com/123.html">this one</a></p>`
	result1 := HTMLToMarkdownWithLinkResolver(html1, resolver)
	assert.Equal(t, "check the post [this one](this.md)", result1)

	// Post 123 (this.md) references post 122 (that.md)
	html2 := `<p>check the previous post <a href="https://can3p_test.livejournal.com/122.html">that one</a></p>`
	result2 := HTMLToMarkdownWithLinkResolver(html2, resolver)
	assert.Equal(t, "check the previous post [that one](that.md)", result2)
}

func TestHTMLToMarkdownWithLinkResolver_EmptyMapping(t *testing.T) {
	// Empty mapping should preserve all links
	resolver := BuildLinkResolver([]PostURLMapping{})

	html := `<p><a href="https://user.livejournal.com/100.html">post</a></p>`
	result := HTMLToMarkdownWithLinkResolver(html, resolver)
	assert.Equal(t, "[post](https://user.livejournal.com/100.html)", result)
}

func TestHTMLToMarkdownWithLinkResolver_MalformedURLs(t *testing.T) {
	urlToFilename := map[string]string{
		"https://user.livejournal.com/100.html": "post.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "relative URL preserved",
			html:     `<p><a href="/100.html">relative</a></p>`,
			expected: "[relative](/100.html)",
		},
		{
			name:     "javascript URL preserved",
			html:     `<p><a href="javascript:void(0)">js link</a></p>`,
			expected: "[js link](javascript:void(0))",
		},
		{
			name:     "mailto URL preserved",
			html:     `<p><a href="mailto:test@example.com">email</a></p>`,
			expected: "[email](mailto:test@example.com)",
		},
		{
			name:     "empty href",
			html:     `<p><a href="">empty</a></p>`,
			expected: "[empty]()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLToMarkdownWithLinkResolver(tt.html, resolver)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLToMarkdownWithLinkResolver_ImageLinks(t *testing.T) {
	// Images wrapped in links should preserve both
	urlToFilename := map[string]string{
		"https://user.livejournal.com/100.html": "post.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	// Note: This is a complex case - an image inside a link
	// The current implementation will output the image as link text
	html := `<p><a href="https://user.livejournal.com/100.html"><img src="https://example.com/img.jpg" alt="thumbnail"></a></p>`
	result := HTMLToMarkdownWithLinkResolver(html, resolver)
	// The image becomes the link text
	assert.Contains(t, result, "![thumbnail](https://example.com/img.jpg)")
	assert.Contains(t, result, "post.md")
}
