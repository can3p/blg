package livejournal

import (
	"testing"

	"github.com/can3p/blg/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreparePost_BasicFields(t *testing.T) {
	c := &client{}

	fields := map[string]string{
		"title":   "Test Post",
		"privacy": "public",
	}
	body := "Hello world!"

	post, images, err := c.PreparePost(fields, body)
	require.NoError(t, err)
	assert.NotNil(t, post)
	assert.Empty(t, images)
	assert.Equal(t, "Test Post", post.Headers["title"])
	assert.Equal(t, "public", post.Headers["security"])
}

func TestPreparePost_PrivacyMapping(t *testing.T) {
	tests := []struct {
		name         string
		privacy      string
		expectedSec  string
		expectedMask any
		expectError  bool
	}{
		{
			name:        "public",
			privacy:     "public",
			expectedSec: "public",
		},
		{
			name:        "private",
			privacy:     "private",
			expectedSec: "private",
		},
		{
			name:         "friends",
			privacy:      "friends",
			expectedSec:  "usemask",
			expectedMask: 1,
		},
		{
			name:        "invalid",
			privacy:     "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{}
			fields := map[string]string{
				"privacy": tt.privacy,
			}

			post, _, err := c.PreparePost(fields, "body")

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedSec, post.Headers["security"])
			if tt.expectedMask != nil {
				assert.Equal(t, tt.expectedMask, post.Headers["allowmask"])
			}
		})
	}
}

func TestPreparePost_OptionalFields(t *testing.T) {
	c := &client{}

	fields := map[string]string{
		"title":    "Test",
		"privacy":  "public",
		"tags":     "coding, go",
		"music":    "Pink Floyd - Time",
		"mood":     "happy",
		"location": "Amsterdam",
		"journal":  "community_name",
	}

	post, _, err := c.PreparePost(fields, "body")
	require.NoError(t, err)

	props, ok := post.Headers["props"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "coding, go", props["taglist"])
	assert.Equal(t, "Pink Floyd - Time", props["current_music"])
	assert.Equal(t, "happy", props["current_mood"])
	assert.Equal(t, "Amsterdam", props["current_location"])
	assert.Equal(t, "community_name", post.Headers["usejournal"])
}

func TestPreparePost_Draft(t *testing.T) {
	c := &client{}

	fields := map[string]string{
		"title": "Draft Post",
		"draft": "yes",
	}

	_, _, err := c.PreparePost(fields, "body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "draft")
}

func TestPreparePost_ExtractImages(t *testing.T) {
	c := &client{}

	fields := map[string]string{
		"title": "Post with images",
	}
	body := "![photo](image.jpg)\n\nSome text\n\n![another](pic.png)"

	_, images, err := c.PreparePost(fields, body)
	require.NoError(t, err)
	assert.Len(t, images, 2)
	assert.Contains(t, images, "image.jpg")
	assert.Contains(t, images, "pic.png")
}

func TestNewPostTemplate(t *testing.T) {
	c := &client{}

	template := c.NewPostTemplate("My New Post")
	assert.Contains(t, template, "title: My New Post")
	assert.Contains(t, template, "privacy: public")
	assert.Contains(t, template, "tags:")
}

func TestExtractItemID(t *testing.T) {
	tests := []struct {
		name        string
		remoteID    string
		expected    int
		expectError bool
	}{
		{
			name:     "numeric ID",
			remoteID: "12345",
			expected: 12345,
		},
		{
			name:     "URL with ditemid",
			remoteID: "https://user.livejournal.com/67890.html",
			expected: 67890,
		},
		{
			name:     "URL without .html",
			remoteID: "https://user.livejournal.com/11111",
			expected: 11111,
		},
		{
			name:        "invalid string",
			remoteID:    "not-a-number",
			expectError: true,
		},
		{
			name:        "invalid URL",
			remoteID:    "https://example.com/invalid.html",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractItemID(tt.remoteID)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPostURL(t *testing.T) {
	c := &client{
		username: "testuser",
		cfg:      types.Config{Host: "www.livejournal.com"},
	}

	// URL passthrough
	url := "https://testuser.livejournal.com/12345.html"
	assert.Equal(t, url, c.PostURL(url))

	// Numeric ID construction
	assert.Contains(t, c.PostURL("67890"), "67890.html")
}

func TestFormatRemotePost(t *testing.T) {
	c := &client{}

	remote := &types.RemotePost{
		ID:        "12345",
		UpdatedAt: 1704067200, // 2024-01-01 00:00:00 UTC
		Data: map[string]any{
			"subject":   "Test Subject",
			"event":     "Post body content",
			"security":  "private",
			"eventtime": "2024-01-01 12:00:00",
			"props": map[string]any{
				"taglist":       "tag1, tag2",
				"current_music": "Some Song",
			},
		},
	}

	fname, content, err := c.FormatRemotePost(remote)
	require.NoError(t, err)

	assert.Contains(t, fname, "2024-01-01")
	assert.Contains(t, fname, ".md")

	contentStr := string(content)
	assert.Contains(t, contentStr, "title: Test Subject")
	assert.Contains(t, contentStr, "privacy: private")
	assert.Contains(t, contentStr, "tags: tag1, tag2")
	assert.Contains(t, contentStr, "music: Some Song")
	assert.Contains(t, contentStr, "Post body content")
}
