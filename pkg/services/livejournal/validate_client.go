//go:build ignore

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/can3p/blg/pkg/services/livejournal"
)

const ljEndpoint = "https://www.livejournal.com/interface/xmlrpc"

// ljClient wraps the XMLRPCClient with credentials
type ljClient struct {
	rpc      *livejournal.XMLRPCClient
	username string
	password string
}

func newLJClient(username, password string) *ljClient {
	return &ljClient{
		rpc:      livejournal.NewXMLRPCClientWithEndpoint(ljEndpoint),
		username: username,
		password: password,
	}
}

func (c *ljClient) call(method string, params map[string]any) (map[string]any, error) {
	return c.rpc.Call(method, params)
}

func (c *ljClient) addAuth(params map[string]any) (map[string]any, error) {
	return c.rpc.AddAuth(params, c.username, c.password)
}

func (c *ljClient) createPost(subject, event, security string, props map[string]any) (int, string, error) {
	now := time.Now()

	params := map[string]any{
		"event":       event,
		"subject":     subject,
		"lineendings": "unix",
		"year":        now.Year(),
		"mon":         int(now.Month()),
		"day":         now.Day(),
		"hour":        now.Hour(),
		"min":         now.Minute(),
		"security":    security,
	}

	if props != nil {
		params["props"] = props
	}

	params, err := c.addAuth(params)
	if err != nil {
		return 0, "", err
	}

	result, err := c.call("LJ.XMLRPC.postevent", params)
	if err != nil {
		return 0, "", err
	}

	itemID, _ := result["itemid"].(int)
	url, _ := result["url"].(string)

	return itemID, url, nil
}

func (c *ljClient) getPost(itemID int) (map[string]any, error) {
	params := map[string]any{
		"selecttype":  "one",
		"itemid":      itemID,
		"lineendings": "unix",
	}

	params, err := c.addAuth(params)
	if err != nil {
		return nil, err
	}

	result, err := c.call("LJ.XMLRPC.getevents", params)
	if err != nil {
		return nil, err
	}

	events, ok := result["events"].([]any)
	if !ok || len(events) == 0 {
		return nil, fmt.Errorf("no events found")
	}

	return events[0].(map[string]any), nil
}

func (c *ljClient) updatePost(itemID int, subject, event, security string) error {
	now := time.Now()

	params := map[string]any{
		"itemid":      itemID,
		"event":       event,
		"subject":     subject,
		"lineendings": "unix",
		"year":        now.Year(),
		"mon":         int(now.Month()),
		"day":         now.Day(),
		"hour":        now.Hour(),
		"min":         now.Minute(),
		"security":    security,
	}

	params, err := c.addAuth(params)
	if err != nil {
		return err
	}

	_, err = c.call("LJ.XMLRPC.editevent", params)
	return err
}

func (c *ljClient) deletePost(itemID int) error {
	now := time.Now()

	params := map[string]any{
		"itemid":      itemID,
		"event":       "",
		"subject":     "",
		"lineendings": "unix",
		"year":        now.Year(),
		"mon":         int(now.Month()),
		"day":         now.Day(),
		"hour":        now.Hour(),
		"min":         now.Minute(),
	}

	params, err := c.addAuth(params)
	if err != nil {
		return err
	}

	_, err = c.call("LJ.XMLRPC.editevent", params)
	return err
}

// Test results
type testResult struct {
	name    string
	passed  bool
	message string
}

func (r testResult) String() string {
	status := "✓ PASS"
	if !r.passed {
		status = "✗ FAIL"
	}
	return fmt.Sprintf("%s: %s - %s", status, r.name, r.message)
}

func main() {
	username := os.Getenv("LJ_USERNAME")
	password := os.Getenv("LJ_PASSWORD")

	if username == "" || password == "" {
		fmt.Println("Usage: LJ_USERNAME=<user> LJ_PASSWORD=<pass> go run validate_client.go")
		fmt.Println("\nSet LJ_USERNAME and LJ_PASSWORD environment variables to run validation.")
		os.Exit(1)
	}

	client := newLJClient(username, password)
	results := []testResult{}
	createdPosts := []int{} // Track posts to clean up

	defer func() {
		// Cleanup: delete all created posts
		fmt.Println("\n=== Cleanup ===")
		for _, itemID := range createdPosts {
			if err := client.deletePost(itemID); err != nil {
				fmt.Printf("Failed to delete post %d: %v\n", itemID, err)
			} else {
				fmt.Printf("Deleted post %d\n", itemID)
			}
		}
	}()

	fmt.Println("=== LiveJournal Client Validation ===")

	// Test 1: Basic post creation and retrieval
	fmt.Println("--- Test 1: Basic post creation and retrieval ---")
	{
		itemID, url, err := client.createPost(
			"Test Post - Basic",
			"This is a simple test post with plain text content.",
			"private",
			nil,
		)
		if err != nil {
			results = append(results, testResult{"Basic creation", false, err.Error()})
		} else {
			createdPosts = append(createdPosts, itemID)
			fmt.Printf("Created post: itemID=%d, url=%s\n", itemID, url)

			// Retrieve and verify
			post, err := client.getPost(itemID)
			if err != nil {
				results = append(results, testResult{"Basic creation", false, "Failed to retrieve: " + err.Error()})
			} else {
				subject, _ := post["subject"].(string)
				event, _ := post["event"].(string)
				security, _ := post["security"].(string)

				if subject == "Test Post - Basic" && strings.Contains(event, "simple test post") && security == "private" {
					results = append(results, testResult{"Basic creation", true, fmt.Sprintf("Post created and retrieved correctly (itemID=%d)", itemID)})
				} else {
					results = append(results, testResult{"Basic creation", false, fmt.Sprintf("Mismatch: subject=%q, security=%q", subject, security)})
				}
			}
		}
	}

	// Test 2: Markdown content (LJ stores as-is, no conversion)
	fmt.Println("\n--- Test 2: Markdown content handling ---")
	{
		markdownContent := `This is **bold** and *italic* text.

Here's a list:
- Item 1
- Item 2
- Item 3

And a [link](https://example.com).

> A blockquote

` + "```" + `
code block
` + "```"

		itemID, _, err := client.createPost(
			"Test Post - Markdown",
			markdownContent,
			"private",
			nil,
		)
		if err != nil {
			results = append(results, testResult{"Markdown content", false, err.Error()})
		} else {
			createdPosts = append(createdPosts, itemID)

			post, err := client.getPost(itemID)
			if err != nil {
				results = append(results, testResult{"Markdown content", false, "Failed to retrieve: " + err.Error()})
			} else {
				event, _ := post["event"].(string)
				fmt.Printf("Stored content:\n%s\n", event)

				// LJ should store markdown as-is (no HTML conversion on server side)
				hasMarkdown := strings.Contains(event, "**bold**") || strings.Contains(event, "*italic*")
				if hasMarkdown {
					results = append(results, testResult{"Markdown content", true, "Markdown stored as-is (expected behavior)"})
				} else {
					// Check if it was converted to HTML
					hasHTML := strings.Contains(event, "<b>") || strings.Contains(event, "<strong>") || strings.Contains(event, "<em>")
					if hasHTML {
						results = append(results, testResult{"Markdown content", false, "Markdown was converted to HTML (unexpected)"})
					} else {
						results = append(results, testResult{"Markdown content", false, "Content modified unexpectedly: " + event[:min(100, len(event))]})
					}
				}
			}
		}
	}

	// Test 3: Post visibility levels
	fmt.Println("\n--- Test 3: Post visibility levels ---")
	{
		visibilityTests := []struct {
			security    string
			allowmask   int
			expectedSec string
		}{
			{"public", 0, ""}, // LJ returns empty string for public (it's the default)
			{"private", 0, "private"},
			{"usemask", 1, "usemask"}, // friends-only
		}

		allPassed := true
		for _, vt := range visibilityTests {
			params := map[string]any{
				"event":       fmt.Sprintf("Testing %s visibility", vt.security),
				"subject":     fmt.Sprintf("Test - %s", vt.security),
				"lineendings": "unix",
				"year":        time.Now().Year(),
				"mon":         int(time.Now().Month()),
				"day":         time.Now().Day(),
				"hour":        time.Now().Hour(),
				"min":         time.Now().Minute(),
				"security":    vt.security,
			}

			if vt.allowmask > 0 {
				params["allowmask"] = vt.allowmask
			}

			params, _ = client.addAuth(params)
			result, err := client.call("LJ.XMLRPC.postevent", params)
			if err != nil {
				results = append(results, testResult{fmt.Sprintf("Visibility %s", vt.security), false, err.Error()})
				allPassed = false
				continue
			}

			itemID, _ := result["itemid"].(int)
			createdPosts = append(createdPosts, itemID)

			post, err := client.getPost(itemID)
			if err != nil {
				results = append(results, testResult{fmt.Sprintf("Visibility %s", vt.security), false, "Failed to retrieve: " + err.Error()})
				allPassed = false
				continue
			}

			security, _ := post["security"].(string)
			if security == vt.expectedSec {
				fmt.Printf("  %s: OK (stored as %q)\n", vt.security, security)
			} else {
				fmt.Printf("  %s: MISMATCH (expected %q, got %q)\n", vt.security, vt.expectedSec, security)
				allPassed = false
			}
		}

		if allPassed {
			results = append(results, testResult{"Visibility levels", true, "All visibility levels work correctly"})
		} else {
			results = append(results, testResult{"Visibility levels", false, "Some visibility levels failed"})
		}
	}

	// Test 4: Post properties (tags, music, mood, location)
	fmt.Println("\n--- Test 4: Post properties ---")
	{
		props := map[string]any{
			"taglist":          "test, validation, blg",
			"current_music":    "Pink Floyd - Time",
			"current_mood":     "productive",
			"current_location": "Amsterdam",
		}

		itemID, _, err := client.createPost(
			"Test Post - Properties",
			"Testing post properties",
			"private",
			props,
		)
		if err != nil {
			results = append(results, testResult{"Post properties", false, err.Error()})
		} else {
			createdPosts = append(createdPosts, itemID)

			post, err := client.getPost(itemID)
			if err != nil {
				results = append(results, testResult{"Post properties", false, "Failed to retrieve: " + err.Error()})
			} else {
				postProps, _ := post["props"].(map[string]any)
				fmt.Printf("Retrieved props: %+v\n", postProps)

				allMatch := true
				for key, expected := range props {
					actual, _ := postProps[key].(string)
					if actual != expected {
						fmt.Printf("  %s: MISMATCH (expected %q, got %q)\n", key, expected, actual)
						allMatch = false
					} else {
						fmt.Printf("  %s: OK\n", key)
					}
				}

				if allMatch {
					results = append(results, testResult{"Post properties", true, "All properties stored correctly"})
				} else {
					results = append(results, testResult{"Post properties", false, "Some properties didn't match"})
				}
			}
		}
	}

	// Test 5: Link handling (local .md links)
	fmt.Println("\n--- Test 5: Link handling ---")
	{
		// First create a "target" post
		targetID, targetURL, err := client.createPost(
			"Target Post",
			"This is the target post for link testing.",
			"private",
			nil,
		)
		if err != nil {
			results = append(results, testResult{"Link handling", false, "Failed to create target post: " + err.Error()})
		} else {
			createdPosts = append(createdPosts, targetID)

			// Create a post with a link to the target
			// In real usage, the client would replace local.md links with actual URLs
			contentWithLink := fmt.Sprintf("See my [previous post](%s) for context.\n\nAlso check [external](https://example.com).", targetURL)

			itemID, _, err := client.createPost(
				"Test Post - Links",
				contentWithLink,
				"private",
				nil,
			)
			if err != nil {
				results = append(results, testResult{"Link handling", false, err.Error()})
			} else {
				createdPosts = append(createdPosts, itemID)

				post, err := client.getPost(itemID)
				if err != nil {
					results = append(results, testResult{"Link handling", false, "Failed to retrieve: " + err.Error()})
				} else {
					event, _ := post["event"].(string)
					fmt.Printf("Stored content with links:\n%s\n", event)

					// Verify links are preserved
					if strings.Contains(event, targetURL) && strings.Contains(event, "https://example.com") {
						results = append(results, testResult{"Link handling", true, "Links preserved correctly"})
					} else {
						results = append(results, testResult{"Link handling", false, "Links were modified"})
					}
				}
			}
		}
	}

	// Test 6: Post update
	fmt.Println("\n--- Test 6: Post update ---")
	{
		// Create initial post
		itemID, _, err := client.createPost(
			"Test Post - Original",
			"Original content",
			"private",
			nil,
		)
		if err != nil {
			results = append(results, testResult{"Post update", false, "Failed to create: " + err.Error()})
		} else {
			createdPosts = append(createdPosts, itemID)

			// Update it
			err = client.updatePost(itemID, "Test Post - Updated", "Updated content with changes", "private")
			if err != nil {
				results = append(results, testResult{"Post update", false, "Failed to update: " + err.Error()})
			} else {
				// Verify update
				post, err := client.getPost(itemID)
				if err != nil {
					results = append(results, testResult{"Post update", false, "Failed to retrieve after update: " + err.Error()})
				} else {
					subject, _ := post["subject"].(string)
					event, _ := post["event"].(string)

					if subject == "Test Post - Updated" && strings.Contains(event, "Updated content") {
						results = append(results, testResult{"Post update", true, "Post updated correctly"})
					} else {
						results = append(results, testResult{"Post update", false, fmt.Sprintf("Update didn't apply: subject=%q", subject)})
					}
				}
			}
		}
	}

	// Test 7: Sync items
	fmt.Println("\n--- Test 7: Sync items ---")
	{
		params := map[string]any{}
		params, _ = client.addAuth(params)

		result, err := client.call("LJ.XMLRPC.syncitems", params)
		if err != nil {
			results = append(results, testResult{"Sync items", false, err.Error()})
		} else {
			count, _ := result["count"].(int)
			total, _ := result["total"].(int)
			syncItems, _ := result["syncitems"].([]any)

			fmt.Printf("Sync result: count=%d, total=%d, items=%d\n", count, total, len(syncItems))

			if len(syncItems) > 0 {
				// Check format of first item
				item := syncItems[0].(map[string]any)
				itemStr, _ := item["item"].(string)
				action, _ := item["action"].(string)

				if strings.HasPrefix(itemStr, "L-") && (action == "create" || action == "update") {
					results = append(results, testResult{"Sync items", true, fmt.Sprintf("Sync working, found %d items", len(syncItems))})
				} else {
					results = append(results, testResult{"Sync items", false, fmt.Sprintf("Unexpected item format: %+v", item)})
				}
			} else {
				results = append(results, testResult{"Sync items", true, "Sync working (no items to sync)"})
			}
		}
	}

	// Test 8: Special characters and Unicode
	fmt.Println("\n--- Test 8: Special characters and Unicode ---")
	{
		// Use simpler Unicode content to test
		unicodeContent := "Testing Unicode: Привет мир"

		itemID, url, err := client.createPost(
			"Test Post - Unicode",
			unicodeContent,
			"private",
			nil,
		)
		if err != nil {
			results = append(results, testResult{"Unicode support", false, err.Error()})
		} else {
			createdPosts = append(createdPosts, itemID)
			fmt.Printf("Created Unicode post: itemID=%d, url=%s\n", itemID, url)

			// Small delay to ensure post is available
			time.Sleep(500 * time.Millisecond)

			post, err := client.getPost(itemID)
			if err != nil {
				results = append(results, testResult{"Unicode support", false, "Failed to retrieve: " + err.Error()})
			} else {
				subject, _ := post["subject"].(string)
				event, _ := post["event"].(string)

				fmt.Printf("Retrieved subject: %q\n", subject)
				fmt.Printf("Retrieved content: %q\n", event)
				fmt.Printf("Event raw value: %T = %v\n", post["event"], post["event"])

				hasRussian := strings.Contains(event, "Привет") || strings.Contains(event, "мир")

				if hasRussian {
					results = append(results, testResult{"Unicode support", true, "Unicode preserved correctly"})
				} else if event == "" {
					// Check if event is stored differently
					if rawEvent := post["event"]; rawEvent != nil {
						results = append(results, testResult{"Unicode support", false, fmt.Sprintf("Event type: %T, value: %v", rawEvent, rawEvent)})
					} else {
						results = append(results, testResult{"Unicode support", false, "Event is nil"})
					}
				} else {
					results = append(results, testResult{"Unicode support", false, fmt.Sprintf("Unicode not found in: %q", event)})
				}
			}
		}
	}

	// Test 9: HTML to Markdown conversion
	// This test creates an HTML post on LJ, fetches it back, and verifies
	// that HTMLToMarkdown correctly converts it to markdown format.
	// See cl-journal/src/markdownify.lisp for reference implementation.
	fmt.Println("\n--- Test 9: HTML to Markdown conversion ---")
	{
		// Create a post with HTML content (as LJ stores it when created via web UI)
		htmlContent := `<p>This is a <strong>bold</strong> and <em>italic</em> paragraph.</p>

<ul>
<li>List item 1</li>
<li>List item 2</li>
</ul>

<blockquote>A quoted text</blockquote>

<p>Check out <a href="https://example.com">this link</a> for more.</p>`

		itemID, _, err := client.createPost(
			"Test Post - HTML Content",
			htmlContent,
			"private",
			nil,
		)
		if err != nil {
			results = append(results, testResult{"HTML to Markdown", false, err.Error()})
		} else {
			createdPosts = append(createdPosts, itemID)

			post, err := client.getPost(itemID)
			if err != nil {
				results = append(results, testResult{"HTML to Markdown", false, "Failed to retrieve: " + err.Error()})
			} else {
				event, _ := post["event"].(string)
				fmt.Printf("Raw HTML from LJ:\n%s\n", event)

				if event == "" {
					results = append(results, testResult{"HTML to Markdown", false, "Event is empty"})
				} else {
					// Convert HTML to markdown using our HTMLToMarkdown function
					markdown := livejournal.HTMLToMarkdown(event)
					fmt.Printf("Converted to markdown:\n%s\n", markdown)

					// Verify conversion produced expected markdown
					hasMarkdownBold := strings.Contains(markdown, "**bold**")
					hasMarkdownItalic := strings.Contains(markdown, "*italic*")
					hasMarkdownList := strings.Contains(markdown, "- List item")
					hasMarkdownLink := strings.Contains(markdown, "[this link](https://example.com)")
					hasMarkdownQuote := strings.Contains(markdown, "> ")

					// Should NOT have HTML tags after conversion
					hasHTML := strings.Contains(markdown, "<strong>") || strings.Contains(markdown, "<p>") || strings.Contains(markdown, "<em>")

					if hasMarkdownBold && hasMarkdownItalic && !hasHTML {
						details := fmt.Sprintf("bold=%v, italic=%v, list=%v, link=%v, quote=%v",
							hasMarkdownBold, hasMarkdownItalic, hasMarkdownList, hasMarkdownLink, hasMarkdownQuote)
						results = append(results, testResult{"HTML to Markdown", true, "HTML correctly converted: " + details})
					} else if hasHTML {
						results = append(results, testResult{"HTML to Markdown", false, "HTML tags still present after conversion"})
					} else {
						results = append(results, testResult{"HTML to Markdown", false, fmt.Sprintf("Missing expected markdown: bold=%v, italic=%v", hasMarkdownBold, hasMarkdownItalic)})
					}
				}
			}
		}
	}

	// Print summary
	fmt.Println("\n=== Test Summary ===")
	passed := 0
	failed := 0
	for _, r := range results {
		fmt.Println(r)
		if r.passed {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("\nTotal: %d passed, %d failed\n", passed, failed)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
	}
	return 0
}
