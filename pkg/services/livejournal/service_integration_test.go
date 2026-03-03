package livejournal

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/can3p/blg/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullMockLJServer is a more complete mock that tracks state
type fullMockLJServer struct {
	*httptest.Server
	mu          sync.Mutex
	posts       map[int]map[string]any
	nextItemID  int
	challengeID int
}

func newFullMockLJServer() *fullMockLJServer {
	m := &fullMockLJServer{
		posts:       make(map[int]map[string]any),
		nextItemID:  100,
		challengeID: 0,
	}

	m.Server = httptest.NewServer(http.HandlerFunc(m.handleRequest))
	return m
}

func (m *fullMockLJServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	bodyStr := string(body)

	w.Header().Set("Content-Type", "text/xml")

	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case strings.Contains(bodyStr, "getchallenge"):
		m.challengeID++
		challenge := fmt.Sprintf("c0:test:%d:60:random:hash", m.challengeID)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct>
<member><name>challenge</name><value><string>` + challenge + `</string></value></member>
<member><name>server_time</name><value><int>1772497135</int></value></member>
<member><name>expire_time</name><value><int>1772497195</int></value></member>
</struct></value></param></params></methodResponse>`))

	case strings.Contains(bodyStr, "postevent"):
		itemID := m.nextItemID
		m.nextItemID++

		// Extract subject and event from request (simplified parsing)
		subject := extractXMLValue(bodyStr, "subject")
		event := extractXMLValue(bodyStr, "event")
		security := extractXMLValue(bodyStr, "security")

		m.posts[itemID] = map[string]any{
			"itemid":    itemID,
			"subject":   subject,
			"event":     event,
			"security":  security,
			"eventtime": "2026-03-03 01:19:00",
		}

		url := fmt.Sprintf("https://test.livejournal.com/%d.html", itemID*256+58)
		_, _ = w.Write(fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct>
<member><name>itemid</name><value><int>%d</int></value></member>
<member><name>url</name><value><string>%s</string></value></member>
<member><name>anum</name><value><int>58</int></value></member>
<member><name>ditemid</name><value><int>%d</int></value></member>
</struct></value></param></params></methodResponse>`, itemID, url, itemID*256+58))

	case strings.Contains(bodyStr, "editevent"):
		// Extract itemid - this could be a ditemid from URL extraction
		itemIDStr := extractXMLValue(bodyStr, "itemid")
		itemID, _ := strconv.Atoi(itemIDStr)

		// Check if this is a ditemid (from URL) - if so, find the real itemid
		// ditemid = itemid * 256 + anum, so itemid = ditemid / 256
		realItemID := itemID
		if itemID > 255 {
			// This is likely a ditemid, convert back to itemid
			realItemID = itemID / 256
		}

		event := extractXMLValue(bodyStr, "event")
		subject := extractXMLValue(bodyStr, "subject")

		if event == "" && subject == "" {
			// Delete
			delete(m.posts, realItemID)
			_, _ = w.Write(fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct>
<member><name>itemid</name><value><int>%d</int></value></member>
<member><name>anum</name><value><int>58</int></value></member>
</struct></value></param></params></methodResponse>`, realItemID))
		} else {
			// Update
			if post, ok := m.posts[realItemID]; ok {
				post["subject"] = subject
				post["event"] = event
			}
			url := fmt.Sprintf("https://test.livejournal.com/%d.html", realItemID*256+58)
			_, _ = w.Write(fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct>
<member><name>itemid</name><value><int>%d</int></value></member>
<member><name>url</name><value><string>%s</string></value></member>
<member><name>anum</name><value><int>58</int></value></member>
</struct></value></param></params></methodResponse>`, realItemID, url))
		}

	case strings.Contains(bodyStr, "syncitems"):
		// Return all posts as sync items
		var items strings.Builder
		for id := range m.posts {
			if items.Len() > 0 {
				items.WriteString("")
			}
			fmt.Fprintf(&items, `<value><struct>
<member><name>item</name><value><string>L-%d</string></value></member>
<member><name>action</name><value><string>create</string></value></member>
<member><name>time</name><value><string>2026-03-03 00:19:13</string></value></member>
</struct></value>`, id)
		}

		_, _ = w.Write(fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct>
<member><name>syncitems</name><value><array><data>%s</data></array></value></member>
<member><name>count</name><value><int>%d</int></value></member>
<member><name>total</name><value><int>%d</int></value></member>
</struct></value></param></params></methodResponse>`, items.String(), len(m.posts), len(m.posts)))

	case strings.Contains(bodyStr, "getevents"):
		// Return requested posts
		var events strings.Builder
		for id, post := range m.posts {
			subject, _ := post["subject"].(string)
			event, _ := post["event"].(string)
			security, _ := post["security"].(string)
			if security == "" {
				security = "public"
			}

			fmt.Fprintf(&events, `<value><struct>
<member><name>itemid</name><value><int>%d</int></value></member>
<member><name>subject</name><value><string>%s</string></value></member>
<member><name>event</name><value><string>%s</string></value></member>
<member><name>security</name><value><string>%s</string></value></member>
<member><name>eventtime</name><value><string>2026-03-03 01:19:00</string></value></member>
<member><name>url</name><value><string>https://test.livejournal.com/%d.html</string></value></member>
</struct></value>`, id, subject, event, security, id*256+58)
		}

		_, _ = w.Write(fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct>
<member><name>events</name><value><array><data>%s</data></array></value></member>
<member><name>lastsync</name><value><string>2026-03-03 00:19:13</string></value></member>
</struct></value></param></params></methodResponse>`, events.String()))

	default:
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><fault><value><struct>
<member><name>faultCode</name><value><int>100</int></value></member>
<member><name>faultString</name><value><string>Unknown method</string></value></member>
</struct></value></fault></methodResponse>`))
	}
}

// extractXMLValue extracts a simple string value from XML-RPC request
func extractXMLValue(xml, name string) string {
	// Look for <name>NAME</name>\n...<value>\n<string>VALUE</string>
	nameTag := "<name>" + name + "</name>"
	_, after, ok := strings.Cut(xml, nameTag)
	if !ok {
		return ""
	}

	// Find the value after this name
	rest := after

	// Look for <string> or <int>
	stringStart := strings.Index(rest, "<string>")
	intStart := strings.Index(rest, "<int>")

	if stringStart != -1 && (intStart == -1 || stringStart < intStart) {
		start := stringStart + len("<string>")
		end := strings.Index(rest[start:], "</string>")
		if end != -1 {
			return rest[start : start+end]
		}
	}

	if intStart != -1 {
		start := intStart + len("<int>")
		end := strings.Index(rest[start:], "</int>")
		if end != -1 {
			return rest[start : start+end]
		}
	}

	return ""
}

func newTestClient(serverURL string) *client {
	return &client{
		cfg:      types.Config{Host: "test.livejournal.com"},
		rpc:      newXMLRPCClient(serverURL),
		username: "testuser",
		password: "testpass",
	}
}

func TestClient_Create(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	post := &types.Post{
		Headers: types.PostHeaders{
			"title":    "Test Post Title",
			"security": "public",
		},
	}

	body, _ := parseBody("This is the post body content.")
	post.Body = body

	remoteID, err := c.Create(post)

	require.NoError(t, err)
	assert.NotEmpty(t, remoteID)
	assert.Contains(t, remoteID, "https://test.livejournal.com/")
	assert.Contains(t, remoteID, ".html")

	// Verify post was stored
	server.mu.Lock()
	assert.Len(t, server.posts, 1)
	server.mu.Unlock()
}

func TestClient_Create_WithProps(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	post := &types.Post{
		Headers: types.PostHeaders{
			"title":    "Post with Props",
			"security": "private",
			"props": map[string]any{
				"taglist":       "go, testing",
				"current_music": "Test Song",
			},
		},
	}

	body, _ := parseBody("Body with props")
	post.Body = body

	remoteID, err := c.Create(post)

	require.NoError(t, err)
	assert.NotEmpty(t, remoteID)
}

func TestClient_Update(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	// First create a post
	post := &types.Post{
		Headers: types.PostHeaders{
			"title":    "Original Title",
			"security": "public",
		},
	}
	body, _ := parseBody("Original content")
	post.Body = body

	remoteID, err := c.Create(post)
	require.NoError(t, err)

	// Now update it
	updatedPost := &types.Post{
		Headers: types.PostHeaders{
			"title":    "Updated Title",
			"security": "public",
		},
	}
	updatedBody, _ := parseBody("Updated content")
	updatedPost.Body = updatedBody

	err = c.Update(remoteID, updatedPost)
	require.NoError(t, err)
}

func TestClient_Update_WithNumericID(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	// Pre-populate a post
	server.mu.Lock()
	server.posts[100] = map[string]any{
		"itemid":  100,
		"subject": "Original",
		"event":   "Original content",
	}
	server.mu.Unlock()

	post := &types.Post{
		Headers: types.PostHeaders{
			"title":    "Updated",
			"security": "public",
		},
	}
	body, _ := parseBody("Updated content")
	post.Body = body

	err := c.Update("100", post)
	require.NoError(t, err)
}

func TestClient_Delete(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	// First create a post
	post := &types.Post{
		Headers: types.PostHeaders{
			"title":    "To Be Deleted",
			"security": "public",
		},
	}
	body, _ := parseBody("This will be deleted")
	post.Body = body

	remoteID, err := c.Create(post)
	require.NoError(t, err)

	server.mu.Lock()
	initialCount := len(server.posts)
	server.mu.Unlock()
	assert.Equal(t, 1, initialCount)

	// Delete it
	err = c.Delete(remoteID)
	require.NoError(t, err)

	server.mu.Lock()
	finalCount := len(server.posts)
	server.mu.Unlock()
	assert.Equal(t, 0, finalCount)
}

func TestClient_Delete_WithNumericID(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	// Pre-populate a post
	server.mu.Lock()
	server.posts[100] = map[string]any{
		"itemid":  100,
		"subject": "To delete",
		"event":   "Content",
	}
	server.mu.Unlock()

	err := c.Delete("100")
	require.NoError(t, err)

	server.mu.Lock()
	_, exists := server.posts[100]
	server.mu.Unlock()
	assert.False(t, exists)
}

func TestClient_FetchPosts(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	// Pre-populate some posts
	server.mu.Lock()
	server.posts[100] = map[string]any{
		"itemid":    100,
		"subject":   "First Post",
		"event":     "First content",
		"security":  "public",
		"eventtime": "2026-03-03 01:19:00",
	}
	server.posts[101] = map[string]any{
		"itemid":    101,
		"subject":   "Second Post",
		"event":     "Second content",
		"security":  "private",
		"eventtime": "2026-03-03 02:00:00",
	}
	server.mu.Unlock()

	posts, imageURLs, err := c.FetchPosts(0)

	require.NoError(t, err)
	assert.Len(t, posts, 2)
	assert.Empty(t, imageURLs) // No images in test posts

	// Verify post data
	foundFirst := false
	foundSecond := false
	for _, p := range posts {
		data := p.Data.(map[string]any)
		subject := data["subject"].(string)
		if subject == "First Post" {
			foundFirst = true
			assert.Equal(t, "public", data["security"])
		}
		if subject == "Second Post" {
			foundSecond = true
			assert.Equal(t, "private", data["security"])
		}
	}
	assert.True(t, foundFirst, "First post not found")
	assert.True(t, foundSecond, "Second post not found")
}

func TestClient_FetchPosts_Empty(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	posts, imageURLs, err := c.FetchPosts(0)

	require.NoError(t, err)
	assert.Empty(t, posts)
	assert.Empty(t, imageURLs)
}

func TestClient_FullWorkflow(t *testing.T) {
	server := newFullMockLJServer()
	defer server.Close()

	c := newTestClient(server.URL)

	// 1. Create a post
	post := &types.Post{
		Headers: types.PostHeaders{
			"title":    "Integration Test Post",
			"security": "public",
		},
	}
	body, _ := parseBody("This is an integration test post.\n\nWith multiple paragraphs.")
	post.Body = body

	remoteID, err := c.Create(post)
	require.NoError(t, err)
	assert.NotEmpty(t, remoteID)

	// 2. Fetch posts and verify
	posts, _, err := c.FetchPosts(0)
	require.NoError(t, err)
	assert.Len(t, posts, 1)

	// 3. Update the post
	updatedPost := &types.Post{
		Headers: types.PostHeaders{
			"title":    "Updated Integration Test Post",
			"security": "private",
		},
	}
	updatedBody, _ := parseBody("This post has been updated.")
	updatedPost.Body = updatedBody

	err = c.Update(remoteID, updatedPost)
	require.NoError(t, err)

	// 4. Delete the post
	err = c.Delete(remoteID)
	require.NoError(t, err)

	// 5. Verify deletion
	posts, _, err = c.FetchPosts(0)
	require.NoError(t, err)
	assert.Empty(t, posts)
}

func TestClient_PostURL(t *testing.T) {
	c := &client{
		username: "testuser",
		cfg:      types.Config{Host: "www.livejournal.com"},
	}

	// URL passthrough
	url := "https://testuser.livejournal.com/12345.html"
	assert.Equal(t, url, c.PostURL(url))

	// HTTP URL passthrough
	httpURL := "http://testuser.livejournal.com/12345.html"
	assert.Equal(t, httpURL, c.PostURL(httpURL))

	// Numeric ID construction
	result := c.PostURL("67890")
	assert.Contains(t, result, "testuser")
	assert.Contains(t, result, "67890.html")
}

func TestClient_UploadImage_NotSupported(t *testing.T) {
	c := &client{}

	_, err := c.UploadImage("test.jpg")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support native image uploads")
}
