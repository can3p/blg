package livejournal

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Real responses captured from LiveJournal API on 2026-03-03

const getChallengeResponse = `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct><member><name>auth_scheme</name><value><string>c0</string></value></member><member><name>server_time</name><value><int>1772497135</int></value></member><member><name>challenge</name><value><string>c0:1772496000:1135:60:MAF2G9b5JvIq6M42Y65Q:df3b924ffb895ac200fa406c3c95714a</string></value></member><member><name>expire_time</name><value><int>1772497195</int></value></member></struct></value></param></params></methodResponse>`

const postEventResponse = `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct><member><name>itemid</name><value><int>71</int></value></member><member><name>url</name><value><string>https://can3p-test.livejournal.com/18234.html</string></value></member><member><name>anum</name><value><int>58</int></value></member><member><name>ditemid</name><value><int>18234</int></value></member></struct></value></param></params></methodResponse>`

const syncItemsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct><member><name>syncitems</name><value><array><data><value><struct><member><name>item</name><value><string>L-71</string></value></member><member><name>action</name><value><string>create</string></value></member><member><name>time</name><value><string>2026-03-03 00:19:13</string></value></member></struct></value></data></array></value></member><member><name>count</name><value><int>1</int></value></member><member><name>total</name><value><int>1</int></value></member></struct></value></param></params></methodResponse>`

const getEventsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct><member><name>skip</name><value><int>0</int></value></member><member><name>events</name><value><array><data><value><struct><member><name>itemid</name><value><int>71</int></value></member><member><name>subject</name><value><string>BLG Integration Test Post</string></value></member><member><name>event</name><value><string>This is a test post from blg integration test.

It will be deleted shortly.</string></value></member><member><name>ditemid</name><value><int>18234</int></value></member><member><name>eventtime</name><value><string>2026-03-03 01:19:00</string></value></member><member><name>props</name><value><struct><member><name>taglist</name><value><string>test, blg</string></value></member></struct></value></member><member><name>can_comment</name><value><int>1</int></value></member><member><name>logtime</name><value><string>2026-03-03 00:19:12</string></value></member><member><name>security</name><value><string>private</string></value></member><member><name>anum</name><value><int>58</int></value></member><member><name>url</name><value><string>https://can3p-test.livejournal.com/18234.html</string></value></member><member><name>event_timestamp</name><value><int>1772500740</int></value></member><member><name>reply_count</name><value><int>0</int></value></member></struct></value></data></array></value></member><member><name>lastsync</name><value><string>2026-03-03 00:19:13</string></value></member></struct></value></param></params></methodResponse>`

const editEventResponse = `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct><member><name>itemid</name><value><int>71</int></value></member><member><name>url</name><value><string>https://can3p-test.livejournal.com/18234.html</string></value></member><member><name>anum</name><value><int>58</int></value></member><member><name>ditemid</name><value><int>18234</int></value></member></struct></value></param></params></methodResponse>`

const deleteEventResponse = `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><params><param><value><struct><member><name>itemid</name><value><int>71</int></value></member><member><name>anum</name><value><int>58</int></value></member></struct></value></param></params></methodResponse>`

const faultResponse = `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse><fault><value><struct><member><name>faultCode</name><value><int>101</int></value></member><member><name>faultString</name><value><string>Invalid password</string></value></member></struct></value></fault></methodResponse>`

func TestXMLRPCClient_Call_GetChallenge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "text/xml", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "LJ.XMLRPC.getchallenge")

		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(getChallengeResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	result, err := client.Call("LJ.XMLRPC.getchallenge", map[string]any{})

	require.NoError(t, err)
	assert.Equal(t, "c0:1772496000:1135:60:MAF2G9b5JvIq6M42Y65Q:df3b924ffb895ac200fa406c3c95714a", result["challenge"])
	assert.Equal(t, "c0", result["auth_scheme"])
	assert.Equal(t, 1772497135, result["server_time"])
	assert.Equal(t, 1772497195, result["expire_time"])
}

func TestXMLRPCClient_Call_PostEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)

		assert.Contains(t, bodyStr, "LJ.XMLRPC.postevent")
		assert.Contains(t, bodyStr, "username")
		assert.Contains(t, bodyStr, "event")

		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(postEventResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	params := map[string]any{
		"username":       "testuser",
		"auth_method":    "challenge",
		"auth_challenge": "test-challenge",
		"auth_response":  "test-response",
		"ver":            1,
		"event":          "Test post body",
		"subject":        "Test Subject",
		"lineendings":    "unix",
		"year":           2026,
		"mon":            3,
		"day":            3,
		"hour":           1,
		"min":            19,
	}

	result, err := client.Call("LJ.XMLRPC.postevent", params)

	require.NoError(t, err)
	assert.Equal(t, 71, result["itemid"])
	assert.Equal(t, "https://can3p-test.livejournal.com/18234.html", result["url"])
	assert.Equal(t, 58, result["anum"])
	assert.Equal(t, 18234, result["ditemid"])
}

func TestXMLRPCClient_Call_SyncItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "LJ.XMLRPC.syncitems")

		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(syncItemsResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	result, err := client.Call("LJ.XMLRPC.syncitems", map[string]any{
		"username": "testuser",
		"ver":      1,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result["count"])
	assert.Equal(t, 1, result["total"])

	syncItems, ok := result["syncitems"].([]any)
	require.True(t, ok)
	require.Len(t, syncItems, 1)

	item := syncItems[0].(map[string]any)
	assert.Equal(t, "L-71", item["item"])
	assert.Equal(t, "create", item["action"])
}

func TestXMLRPCClient_Call_GetEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "LJ.XMLRPC.getevents")

		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(getEventsResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	result, err := client.Call("LJ.XMLRPC.getevents", map[string]any{
		"username":   "testuser",
		"ver":        1,
		"selecttype": "one",
		"itemid":     71,
	})

	require.NoError(t, err)

	events, ok := result["events"].([]any)
	require.True(t, ok)
	require.Len(t, events, 1)

	event := events[0].(map[string]any)
	assert.Equal(t, 71, event["itemid"])
	assert.Equal(t, "BLG Integration Test Post", event["subject"])
	assert.Contains(t, event["event"], "test post from blg")
	assert.Equal(t, "private", event["security"])
	assert.Equal(t, "2026-03-03 01:19:00", event["eventtime"])

	props := event["props"].(map[string]any)
	assert.Equal(t, "test, blg", props["taglist"])
}

func TestXMLRPCClient_Call_EditEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "LJ.XMLRPC.editevent")

		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(editEventResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	result, err := client.Call("LJ.XMLRPC.editevent", map[string]any{
		"username": "testuser",
		"ver":      1,
		"itemid":   71,
		"event":    "Updated content",
		"subject":  "Updated Subject",
	})

	require.NoError(t, err)
	assert.Equal(t, 71, result["itemid"])
	assert.Equal(t, "https://can3p-test.livejournal.com/18234.html", result["url"])
}

func TestXMLRPCClient_Call_DeleteEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		assert.Contains(t, bodyStr, "LJ.XMLRPC.editevent")
		// Delete sends empty event and subject
		assert.Contains(t, bodyStr, "<name>event</name>")

		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(deleteEventResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	result, err := client.Call("LJ.XMLRPC.editevent", map[string]any{
		"username": "testuser",
		"ver":      1,
		"itemid":   71,
		"event":    "",
		"subject":  "",
	})

	require.NoError(t, err)
	assert.Equal(t, 71, result["itemid"])
	// Delete response doesn't include URL
	_, hasURL := result["url"]
	assert.False(t, hasURL)
}

func TestXMLRPCClient_Call_Fault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(faultResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	_, err := client.Call("LJ.XMLRPC.postevent", map[string]any{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "XML-RPC fault")
	assert.Contains(t, err.Error(), "101")
	assert.Contains(t, err.Error(), "Invalid password")
}

func TestXMLRPCClient_GetChallenge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(getChallengeResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	challenge, err := client.GetChallenge()

	require.NoError(t, err)
	assert.Equal(t, "c0:1772496000:1135:60:MAF2G9b5JvIq6M42Y65Q:df3b924ffb895ac200fa406c3c95714a", challenge)
}

func TestXMLRPCClient_AddAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(getChallengeResponse))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	params := map[string]any{"event": "test"}

	result, err := client.AddAuth(params, "testuser", "testpass")

	require.NoError(t, err)
	assert.Equal(t, "testuser", result["username"])
	assert.Equal(t, "challenge", result["auth_method"])
	assert.Equal(t, "c0:1772496000:1135:60:MAF2G9b5JvIq6M42Y65Q:df3b924ffb895ac200fa406c3c95714a", result["auth_challenge"])
	assert.NotEmpty(t, result["auth_response"])
	assert.Equal(t, 1, result["ver"])
	// Original param preserved
	assert.Equal(t, "test", result["event"])
}

func TestDecodeResponse_RealXML(t *testing.T) {
	// Test decoding actual XML responses from LJ

	t.Run("getchallenge", func(t *testing.T) {
		result, err := decodeResponse([]byte(getChallengeResponse))
		require.NoError(t, err)
		assert.Equal(t, "c0:1772496000:1135:60:MAF2G9b5JvIq6M42Y65Q:df3b924ffb895ac200fa406c3c95714a", result["challenge"])
	})

	t.Run("postevent", func(t *testing.T) {
		result, err := decodeResponse([]byte(postEventResponse))
		require.NoError(t, err)
		assert.Equal(t, 71, result["itemid"])
		assert.Equal(t, "https://can3p-test.livejournal.com/18234.html", result["url"])
	})

	t.Run("syncitems", func(t *testing.T) {
		result, err := decodeResponse([]byte(syncItemsResponse))
		require.NoError(t, err)
		items := result["syncitems"].([]any)
		assert.Len(t, items, 1)
	})

	t.Run("getevents", func(t *testing.T) {
		result, err := decodeResponse([]byte(getEventsResponse))
		require.NoError(t, err)
		events := result["events"].([]any)
		assert.Len(t, events, 1)
		event := events[0].(map[string]any)
		assert.Equal(t, "BLG Integration Test Post", event["subject"])
	})

	t.Run("fault", func(t *testing.T) {
		_, err := decodeResponse([]byte(faultResponse))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid password")
	})
}

func TestEncodeRequest_XMLStructure(t *testing.T) {
	params := map[string]any{
		"username": "testuser",
		"ver":      1,
		"props": map[string]any{
			"taglist": "tag1, tag2",
		},
	}

	data, err := encodeRequest("LJ.XMLRPC.postevent", params)
	require.NoError(t, err)

	// Verify it's valid XML
	var call xmlrpcMethodCall
	err = xml.Unmarshal(data, &call)
	require.NoError(t, err)

	assert.Equal(t, "LJ.XMLRPC.postevent", call.MethodName)
	assert.Len(t, call.Params, 1)

	// Verify struct is properly encoded
	xmlStr := string(data)
	assert.Contains(t, xmlStr, "<name>username</name>")
	assert.Contains(t, xmlStr, "<string>testuser</string>")
	assert.Contains(t, xmlStr, "<name>ver</name>")
	assert.Contains(t, xmlStr, "<int>1</int>")
	assert.Contains(t, xmlStr, "<name>props</name>")
	assert.Contains(t, xmlStr, "<name>taglist</name>")
}

func TestXMLRoundTrip(t *testing.T) {
	// Test that we can encode params and decode a response correctly

	params := map[string]any{
		"username": "test",
		"ver":      1,
		"items":    []any{"a", "b"},
		"nested": map[string]any{
			"key": "value",
		},
	}

	encoded, err := encodeRequest("test.method", params)
	require.NoError(t, err)

	// Verify XML is parseable
	var call xmlrpcMethodCall
	err = xml.Unmarshal(encoded, &call)
	require.NoError(t, err)
	assert.Equal(t, "test.method", call.MethodName)
}

func TestXMLRPCClient_NetworkError(t *testing.T) {
	// Test handling of network errors
	client := newXMLRPCClient("http://localhost:1") // Invalid port

	_, err := client.Call("test", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "executing request")
}

func TestXMLRPCClient_InvalidXMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not valid xml"))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	_, err := client.Call("test", map[string]any{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshaling response")
}

func TestXMLRPCClient_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<?xml version="1.0"?><methodResponse><params></params></methodResponse>`))
	}))
	defer server.Close()

	client := newXMLRPCClient(server.URL)
	_, err := client.Call("test", map[string]any{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}
