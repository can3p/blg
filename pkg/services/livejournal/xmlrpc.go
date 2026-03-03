package livejournal

import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"
)

type xmlrpcClient struct {
	endpoint   string
	httpClient *http.Client
}

func newXMLRPCClient(host string) *xmlrpcClient {
	return &xmlrpcClient{
		endpoint: host + endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
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

func (c *xmlrpcClient) call(method string, params map[string]any) (map[string]any, error) {
	reqBody, err := encodeRequest(method, params)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return decodeResponse(respBody)
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
		return *v.Base64
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

func md5Hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (c *xmlrpcClient) getChallenge() (string, error) {
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

func (c *xmlrpcClient) addAuth(params map[string]any, username, password string) (map[string]any, error) {
	challenge, err := c.getChallenge()
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
