package livejournal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBody(t *testing.T) {
	body := "Hello **world**!"
	parser, err := parseBody(body)
	require.NoError(t, err)
	assert.NotNil(t, parser)
}

func TestMaybeString(t *testing.T) {
	body := "Hello **world**!"
	parser, err := parseBody(body)
	require.NoError(t, err)

	result, err := parser.MaybeString()
	require.NoError(t, err)
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "world")
}

func TestExtractImages(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected []string
	}{
		{
			name:     "no images",
			body:     "Just some text",
			expected: []string{},
		},
		{
			name:     "single image",
			body:     "![alt](image.jpg)",
			expected: []string{"image.jpg"},
		},
		{
			name:     "multiple images",
			body:     "![one](a.jpg)\n\nSome text\n\n![two](b.png)",
			expected: []string{"a.jpg", "b.png"},
		},
		{
			name:     "image with URL",
			body:     "![photo](https://example.com/photo.jpg)",
			expected: []string{"https://example.com/photo.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := parseBody(tt.body)
			require.NoError(t, err)

			images, err := parser.ExtractImages()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, images)
		})
	}
}

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected []string
	}{
		{
			name:     "no links",
			body:     "Just some text",
			expected: []string{},
		},
		{
			name:     "external link - not extracted",
			body:     "[Google](https://google.com)",
			expected: []string{},
		},
		{
			name:     "local md link",
			body:     "[previous post](2024-01-01-other.md)",
			expected: []string{"2024-01-01-other.md"},
		},
		{
			name:     "multiple local links",
			body:     "[one](a.md) and [two](b.md)",
			expected: []string{"a.md", "b.md"},
		},
		{
			name:     "mixed links",
			body:     "[local](post.md) and [external](https://example.com)",
			expected: []string{"post.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := parseBody(tt.body)
			require.NoError(t, err)

			links, err := parser.ExtractLinks()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, links)
		})
	}
}

func TestReplaceImages(t *testing.T) {
	body := "![photo](local.jpg)"
	parser, err := parseBody(body)
	require.NoError(t, err)

	err = parser.ReplaceImages(map[string]string{
		"local.jpg": "https://cdn.example.com/abc123.jpg",
	})
	require.NoError(t, err)

	result, err := parser.MaybeString()
	require.NoError(t, err)
	assert.Contains(t, result, "https://cdn.example.com/abc123.jpg")
	assert.NotContains(t, result, "local.jpg")
}

func TestReplaceLinks(t *testing.T) {
	body := "See [my post](2024-01-01-test.md) for details."
	parser, err := parseBody(body)
	require.NoError(t, err)

	err = parser.ReplaceLinks(map[string]string{
		"2024-01-01-test.md": "https://user.livejournal.com/12345.html",
	})
	require.NoError(t, err)

	result, err := parser.MaybeString()
	require.NoError(t, err)
	assert.Contains(t, result, "https://user.livejournal.com/12345.html")
	assert.NotContains(t, result, "2024-01-01-test.md")
}
