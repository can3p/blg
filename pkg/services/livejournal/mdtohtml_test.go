package livejournal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdownToHTML_Basic(t *testing.T) {
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "plain text",
			input:    "Hello world",
			contains: []string{"<p>Hello world</p>"},
		},
		{
			name:     "bold text",
			input:    "This is **bold** text",
			contains: []string{"<strong>bold</strong>"},
		},
		{
			name:     "italic text",
			input:    "This is *italic* text",
			contains: []string{"<em>italic</em>"},
		},
		{
			name:     "link",
			input:    "Check [this](https://example.com)",
			contains: []string{`<a href="https://example.com">this</a>`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkdownToHTML(tt.input, config)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestMarkdownToHTML_YouTubeStandalone(t *testing.T) {
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "youtube.com watch URL",
			input:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expected: `<lj-embed source="youtube" vid="dQw4w9WgXcQ"></lj-embed>`,
		},
		{
			name:     "youtu.be short URL",
			input:    "https://youtu.be/dQw4w9WgXcQ",
			expected: `<lj-embed source="youtube" vid="dQw4w9WgXcQ"></lj-embed>`,
		},
		{
			name:     "youtube embed URL",
			input:    "https://www.youtube.com/embed/dQw4w9WgXcQ",
			expected: `<lj-embed source="youtube" vid="dQw4w9WgXcQ"></lj-embed>`,
		},
		{
			name:     "youtube with extra params",
			input:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=120",
			expected: `<lj-embed source="youtube" vid="dQw4w9WgXcQ"></lj-embed>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkdownToHTML(tt.input, config)
			assert.Contains(t, result, tt.expected)
			// Should NOT contain <p> wrapper for standalone embeds
			assert.NotContains(t, result, "<p>")
		})
	}
}

func TestMarkdownToHTML_YouTubeInText(t *testing.T) {
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	input := "Check out this video: https://www.youtube.com/watch?v=dQw4w9WgXcQ it's great!"

	result := MarkdownToHTML(input, config)

	// Should contain the paragraph with the text
	assert.Contains(t, result, "<p>")
	assert.Contains(t, result, "Check out this video")
	// Should also contain the embed after the paragraph
	assert.Contains(t, result, `<lj-embed source="youtube" vid="dQw4w9WgXcQ"></lj-embed>`)
}

func TestMarkdownToHTML_MultipleYouTubeLinks(t *testing.T) {
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	// Each URL on its own line with blank lines = separate paragraphs = separate embeds
	input := `https://www.youtube.com/watch?v=dQw4w9WgXcQ

https://www.youtube.com/watch?v=9bZkp7q19f0`

	result := MarkdownToHTML(input, config)

	// Should contain both embeds
	assert.Contains(t, result, `vid="dQw4w9WgXcQ"`)
	assert.Contains(t, result, `vid="9bZkp7q19f0"`)
}

func TestMarkdownToHTML_YouTubeLinksInSameParagraph(t *testing.T) {
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	// URLs in same paragraph (no blank line) - both should get embeds appended
	input := `Check these: https://www.youtube.com/watch?v=dQw4w9WgXcQ and https://www.youtube.com/watch?v=9bZkp7q19f0`

	result := MarkdownToHTML(input, config)

	// Should contain both embeds after the paragraph
	assert.Contains(t, result, `vid="dQw4w9WgXcQ"`, "First video ID should be present")
	assert.Contains(t, result, `vid="9bZkp7q19f0"`, "Second video ID should be present")

	// Debug: print the result
	t.Logf("Result: %s", result)
}

func TestYouTubeLinkRegex(t *testing.T) {
	text := "Check these: https://www.youtube.com/watch?v=dQw4w9WgXcQ and https://www.youtube.com/watch?v=9bZkp7q19f0"
	matches := youtubeLinkRE.FindAllStringSubmatch(text, -1)
	t.Logf("Found %d matches: %v", len(matches), matches)
	assert.Equal(t, 2, len(matches), "Should find 2 YouTube URLs")
	if len(matches) >= 2 {
		assert.Equal(t, "dQw4w9WgXcQ", matches[0][1])
		assert.Equal(t, "9bZkp7q19f0", matches[1][1])
	}
}

func TestMarkdownToHTML_UserHandle(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		serviceHost string
		contains    string
	}{
		{
			name:        "livejournal user",
			input:       "Hello @john_doe how are you?",
			serviceHost: "livejournal.com",
			contains:    `<a href="https://john_doe.livejournal.com/">@john_doe</a>`,
		},
		{
			name:        "dreamwidth user",
			input:       "Check out @alice",
			serviceHost: "dreamwidth.org",
			contains:    `<a href="https://alice.dreamwidth.org/">@alice</a>`,
		},
		{
			name:        "multiple users",
			input:       "@alice and @bob are friends",
			serviceHost: "livejournal.com",
			contains:    `@alice</a>`,
		},
		{
			name:        "user with underscore",
			input:       "Hey @user_name_123",
			serviceHost: "livejournal.com",
			contains:    `<a href="https://user_name_123.livejournal.com/">@user_name_123</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ServiceConfig{ServiceHost: tt.serviceHost}
			result := MarkdownToHTML(tt.input, config)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestMarkdownToHTML_UserHandleFallback(t *testing.T) {
	// When no service host is configured, use LJ user tag
	config := ServiceConfig{ServiceHost: ""}
	input := "Hello @john_doe"

	result := MarkdownToHTML(input, config)

	assert.Contains(t, result, `<lj user="john_doe">`)
}

func TestMarkdownToHTML_UserHandleNotMatched(t *testing.T) {
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "email address",
			input: "Contact me at user@example.com",
		},
		{
			name:  "too short username",
			input: "Hey @ab",
		},
		{
			name:  "starts with number",
			input: "Hey @123user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkdownToHTML(tt.input, config)
			// Should not contain user link format
			assert.NotContains(t, result, `.livejournal.com/">@`)
		})
	}
}

func TestMarkdownToHTML_Combined(t *testing.T) {
	config := ServiceConfig{ServiceHost: "livejournal.com"}

	input := `Hey @alice, check out this video:

https://www.youtube.com/watch?v=dQw4w9WgXcQ

What do you think @bob?`

	result := MarkdownToHTML(input, config)

	// Should contain user links
	assert.Contains(t, result, `alice.livejournal.com`)
	assert.Contains(t, result, `bob.livejournal.com`)
	// Should contain YouTube embed
	assert.Contains(t, result, `<lj-embed source="youtube" vid="dQw4w9WgXcQ"></lj-embed>`)
}

func TestYouTubeEmbedCode(t *testing.T) {
	result := YouTubeEmbedCode("dQw4w9WgXcQ")
	assert.Equal(t, `<lj-embed source="youtube" vid="dQw4w9WgXcQ"></lj-embed>`, result)
}

func TestUserLink(t *testing.T) {
	result := UserLink("john_doe", "livejournal.com")
	assert.Equal(t, `<a href="https://john_doe.livejournal.com/">@john_doe</a>`, result)
}

func TestLJUserTag(t *testing.T) {
	result := LJUserTag("john_doe")
	assert.Equal(t, `<lj user="john_doe">`, result)
}
