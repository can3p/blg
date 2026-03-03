package livejournal

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type XMLRPCClient struct {
	endpoint    string
	httpClient  *http.Client
	lastRequest time.Time
	mu          sync.Mutex
	rateLimit   time.Duration
}

func newXMLRPCClient(host string) *XMLRPCClient {
	return &XMLRPCClient{
		endpoint: host + endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: time.Second, // 1 RPS limit for LiveJournal API
	}
}

// NewXMLRPCClientWithEndpoint creates a client with a full endpoint URL (for testing)
func NewXMLRPCClientWithEndpoint(endpointURL string) *XMLRPCClient {
	return &XMLRPCClient{
		endpoint: endpointURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: time.Second, // 1 RPS limit for LiveJournal API
	}
}

type xmlrpcValue struct {
	String  *string       `xml:"string,omitempty"`
	Int     *int          `xml:"int,omitempty"`
	I4      *int          `xml:"i4,omitempty"`
	Boolean *int          `xml:"boolean,omitempty"`
	Struct  *xmlrpcStruct `xml:"struct,omitempty"`
	Array   *xmlrpcArray  `xml:"array,omitempty"`
	Base64  *string       `xml:"base64,omitempty"`
	// RawText captures text content when no type wrapper is present
	// XML-RPC allows <value>text</value> without <string> wrapper
	RawText string `xml:",chardata"`
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

func (c *XMLRPCClient) Call(method string, params map[string]any) (map[string]any, error) {
	reqBody, err := encodeRequest(method, params)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Rate limiting: ensure minimum time between requests
		c.mu.Lock()
		if c.rateLimit > 0 && !c.lastRequest.IsZero() {
			elapsed := time.Since(c.lastRequest)
			if elapsed < c.rateLimit {
				time.Sleep(c.rateLimit - elapsed)
			}
		}
		c.lastRequest = time.Now()
		c.mu.Unlock()

		req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", "text/xml")
		req.Header.Set("User-Agent", "blg/1.0 (https://github.com/can3p/blg)")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			// Exponential backoff: 2s, 4s, 8s
			backoff := time.Duration(2<<attempt) * time.Second
			time.Sleep(backoff)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			backoff := time.Duration(2<<attempt) * time.Second
			time.Sleep(backoff)
			continue
		}

		return decodeResponse(respBody)
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
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

func encodeValue(v any) xmlrpcValue {
	switch val := v.(type) {
	case string:
		return xmlrpcValue{String: &val}
	case int:
		return xmlrpcValue{Int: &val}
	case int64:
		i := int(val)
		return xmlrpcValue{Int: &i}
	case bool:
		b := 0
		if val {
			b = 1
		}
		return xmlrpcValue{Boolean: &b}
	case map[string]any:
		members := make([]xmlrpcMember, 0, len(val))
		// Sort keys for deterministic output
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
	case v.Base64 != nil:
		// Decode base64 content (LJ returns Unicode strings as base64)
		decoded, err := base64.StdEncoding.DecodeString(*v.Base64)
		if err != nil {
			return *v.Base64 // Return raw if decode fails
		}
		return string(decoded)
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
		// XML-RPC allows <value>text</value> without type wrapper
		// In this case, RawText contains the string value
		if v.RawText != "" {
			return strings.TrimSpace(v.RawText)
		}
		return nil
	}
}

func md5Hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (c *XMLRPCClient) GetChallenge() (string, error) {
	result, err := c.Call("LJ.XMLRPC.getchallenge", map[string]any{})
	if err != nil {
		return "", err
	}

	challenge, ok := result["challenge"].(string)
	if !ok {
		return "", fmt.Errorf("challenge not found in response")
	}

	return challenge, nil
}

func (c *XMLRPCClient) AddAuth(params map[string]any, username, password string) (map[string]any, error) {
	challenge, err := c.GetChallenge()
	if err != nil {
		return nil, fmt.Errorf("getting challenge: %w", err)
	}

	authResponse := md5Hash(challenge + md5Hash(password))

	params["username"] = username
	params["auth_method"] = "challenge"
	params["auth_challenge"] = challenge
	params["auth_response"] = authResponse
	params["ver"] = 1

	return params, nil
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
	}
	return 0
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
