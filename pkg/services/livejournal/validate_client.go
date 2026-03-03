//go:build ignore

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const endpoint = "https://www.livejournal.com/interface/xmlrpc"

// XML-RPC types
type xmlrpcValue struct {
	String  *string       `xml:"string,omitempty"`
	Int     *int          `xml:"int,omitempty"`
	I4      *int          `xml:"i4,omitempty"`
	Boolean *int          `xml:"boolean,omitempty"`
	Struct  *xmlrpcStruct `xml:"struct,omitempty"`
	Array   *xmlrpcArray  `xml:"array,omitempty"`
}

type xmlrpcMember struct {
	Name  string      `xml:"name"`
	Value xmlrpcValue `xml:"value"`
}

type xmlrpcStruct struct {
	Members []xmlrpcMember `xml:"member"`
}

type xmlrpcArray struct {
	Data struct {
		Values []xmlrpcValue `xml:"value"`
	} `xml:"data"`
}

type xmlrpcParam struct {
	Value xmlrpcValue `xml:"value"`
}

type xmlrpcMethodCall struct {
	XMLName    xml.Name      `xml:"methodCall"`
	MethodName string        `xml:"methodName"`
	Params     []xmlrpcParam `xml:"params>param"`
}

type xmlrpcMethodResponse struct {
	Params []xmlrpcParam `xml:"params>param"`
	Fault  *struct {
		Value xmlrpcValue `xml:"value"`
	} `xml:"fault"`
}

func encodeValue(v any) xmlrpcValue {
	switch val := v.(type) {
	case string:
		return xmlrpcValue{String: &val}
	case int:
		return xmlrpcValue{Int: &val}
	case bool:
		b := 0
		if val {
			b = 1
		}
		return xmlrpcValue{Boolean: &b}
	case map[string]any:
		members := make([]xmlrpcMember, 0, len(val))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			members = append(members, xmlrpcMember{
				Name:  k,
				Value: encodeValue(val[k]),
			})
		}
		return xmlrpcValue{Struct: &xmlrpcStruct{Members: members}}
	case []any:
		values := make([]xmlrpcValue, len(val))
		for i, item := range val {
			values[i] = encodeValue(item)
		}
		arr := &xmlrpcArray{}
		arr.Data.Values = values
		return xmlrpcValue{Array: arr}
	default:
		s := fmt.Sprintf("%v", val)
		return xmlrpcValue{String: &s}
	}
}

func encodeRequest(method string, params map[string]any) ([]byte, error) {
	call := xmlrpcMethodCall{
		MethodName: method,
		Params:     []xmlrpcParam{{Value: encodeValue(params)}},
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(call); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeValue(v xmlrpcValue) any {
	switch {
	case v.String != nil:
		return *v.String
	case v.Int != nil:
		return *v.Int
	case v.I4 != nil:
		return *v.I4
	case v.Boolean != nil:
		return *v.Boolean != 0
	case v.Struct != nil:
		m := make(map[string]any)
		for _, member := range v.Struct.Members {
			m[member.Name] = decodeValue(member.Value)
		}
		return m
	case v.Array != nil:
		arr := make([]any, len(v.Array.Data.Values))
		for i, val := range v.Array.Data.Values {
			arr[i] = decodeValue(val)
		}
		return arr
	default:
		return nil
	}
}

func decodeResponse(data []byte) (map[string]any, error) {
	var resp xmlrpcMethodResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	if resp.Fault != nil {
		faultMap := decodeValue(resp.Fault.Value)
		if m, ok := faultMap.(map[string]any); ok {
			return nil, fmt.Errorf("XML-RPC fault: %v - %v", m["faultCode"], m["faultString"])
		}
		return nil, fmt.Errorf("XML-RPC fault: %v", faultMap)
	}

	if len(resp.Params) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	result := decodeValue(resp.Params[0].Value)
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}

	return map[string]any{"result": result}, nil
}

func md5Hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

type ljClient struct {
	username string
	password string
	client   *http.Client
}

func newLJClient(username, password string) *ljClient {
	return &ljClient{
		username: username,
		password: password,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *ljClient) call(method string, params map[string]any) (map[string]any, error) {
	reqBody, err := encodeRequest(method, params)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	var respBody []byte
	var lastErr error

	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", "text/xml")
		req.Header.Set("User-Agent", "blg/1.0")

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}

		return decodeResponse(respBody)
	}

	return nil, fmt.Errorf("request failed after retries: %w", lastErr)
}

func (c *ljClient) getChallenge() (string, error) {
	result, err := c.call("LJ.XMLRPC.getchallenge", map[string]any{})
	if err != nil {
		return "", err
	}

	challenge, ok := result["challenge"].(string)
	if !ok {
		return "", fmt.Errorf("challenge not found in response")
	}

	return challenge, nil
}

func (c *ljClient) addAuth(params map[string]any) (map[string]any, error) {
	challenge, err := c.getChallenge()
	if err != nil {
		return nil, fmt.Errorf("getting challenge: %w", err)
	}

	authResponse := md5Hash(challenge + md5Hash(c.password))

	params["username"] = c.username
	params["auth_method"] = "challenge"
	params["auth_challenge"] = challenge
	params["auth_response"] = authResponse
	params["ver"] = 1

	return params, nil
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
	fmt.Println()

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
	// that the HTML has been converted to markdown format.
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

				// TODO: When HTML-to-markdown conversion is implemented in FormatRemotePost,
				// this test should verify the conversion produces:
				// - **bold** instead of <strong>bold</strong>
				// - *italic* or _italic_ instead of <em>italic</em>
				// - * List item instead of <li>List item</li>
				// - > A quoted text instead of <blockquote>
				// - [this link](https://example.com) instead of <a href="...">

				// For now, just verify we can fetch HTML content
				if event == "" {
					results = append(results, testResult{"HTML to Markdown", false, "Event is empty"})
				} else {
					// Check if HTML is present (current behavior - no conversion yet)
					hasHTML := strings.Contains(event, "<strong>") || strings.Contains(event, "<p>") || strings.Contains(event, "<em>")
					// Check if markdown is present (expected after conversion is implemented)
					hasMarkdown := strings.Contains(event, "**bold**") || strings.Contains(event, "*italic*")

					if hasMarkdown && !hasHTML {
						results = append(results, testResult{"HTML to Markdown", true, "HTML correctly converted to markdown"})
					} else if hasHTML {
						results = append(results, testResult{"HTML to Markdown", false, "HTML not converted to markdown (feature not implemented yet)"})
					} else {
						results = append(results, testResult{"HTML to Markdown", false, fmt.Sprintf("Unexpected content: %s", event[:min(100, len(event))])})
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
