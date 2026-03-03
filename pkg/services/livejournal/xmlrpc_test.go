package livejournal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeValue_String(t *testing.T) {
	v := encodeValue("hello")
	require.NotNil(t, v.String)
	assert.Equal(t, "hello", *v.String)
}

func TestEncodeValue_Int(t *testing.T) {
	v := encodeValue(42)
	require.NotNil(t, v.Int)
	assert.Equal(t, 42, *v.Int)
}

func TestEncodeValue_Bool(t *testing.T) {
	v := encodeValue(true)
	require.NotNil(t, v.Boolean)
	assert.Equal(t, 1, *v.Boolean)

	v = encodeValue(false)
	require.NotNil(t, v.Boolean)
	assert.Equal(t, 0, *v.Boolean)
}

func TestEncodeValue_Map(t *testing.T) {
	m := map[string]any{
		"key1": "value1",
		"key2": 123,
	}
	v := encodeValue(m)
	require.NotNil(t, v.Struct)
	assert.Len(t, v.Struct.Members, 2)
}

func TestEncodeValue_Array(t *testing.T) {
	arr := []any{"a", "b", "c"}
	v := encodeValue(arr)
	require.NotNil(t, v.Array)
	assert.Len(t, v.Array.Data.Values, 3)
}

func TestDecodeValue_String(t *testing.T) {
	s := "test"
	v := xmlrpcValue{String: &s}
	result := decodeValue(v)
	assert.Equal(t, "test", result)
}

func TestDecodeValue_Int(t *testing.T) {
	i := 42
	v := xmlrpcValue{Int: &i}
	result := decodeValue(v)
	assert.Equal(t, 42, result)
}

func TestDecodeValue_Boolean(t *testing.T) {
	b := 1
	v := xmlrpcValue{Boolean: &b}
	result := decodeValue(v)
	assert.Equal(t, true, result)

	b = 0
	v = xmlrpcValue{Boolean: &b}
	result = decodeValue(v)
	assert.Equal(t, false, result)
}

func TestDecodeValue_Struct(t *testing.T) {
	s := "value"
	v := xmlrpcValue{
		Struct: &xmlrpcStruct{
			Members: []xmlrpcMember{
				{Name: "key", Value: xmlrpcValue{String: &s}},
			},
		},
	}
	result := decodeValue(v)
	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value", m["key"])
}

func TestDecodeValue_Array(t *testing.T) {
	s1 := "a"
	s2 := "b"
	v := xmlrpcValue{
		Array: &xmlrpcArray{
			Data: struct {
				Values []xmlrpcValue `xml:"value"`
			}{
				Values: []xmlrpcValue{
					{String: &s1},
					{String: &s2},
				},
			},
		},
	}
	result := decodeValue(v)
	arr, ok := result.([]any)
	require.True(t, ok)
	assert.Len(t, arr, 2)
	assert.Equal(t, "a", arr[0])
	assert.Equal(t, "b", arr[1])
}

func TestMd5Hash(t *testing.T) {
	hash := md5Hash("test")
	assert.Equal(t, "098f6bcd4621d373cade4e832627b4f6", hash)
}

func TestGetInt(t *testing.T) {
	m := map[string]any{
		"int":     42,
		"int64":   int64(100),
		"float64": float64(200),
		"string":  "300",
		"invalid": "not a number",
	}

	assert.Equal(t, 42, getInt(m, "int"))
	assert.Equal(t, 100, getInt(m, "int64"))
	assert.Equal(t, 200, getInt(m, "float64"))
	assert.Equal(t, 300, getInt(m, "string"))
	assert.Equal(t, 0, getInt(m, "invalid"))
	assert.Equal(t, 0, getInt(m, "missing"))
}

func TestGetString(t *testing.T) {
	m := map[string]any{
		"str":    "hello",
		"notstr": 123,
	}

	assert.Equal(t, "hello", getString(m, "str"))
	assert.Equal(t, "", getString(m, "notstr"))
	assert.Equal(t, "", getString(m, "missing"))
}

func TestEncodeRequest(t *testing.T) {
	params := map[string]any{
		"username": "testuser",
		"ver":      1,
	}

	data, err := encodeRequest("LJ.XMLRPC.getchallenge", params)
	require.NoError(t, err)

	xml := string(data)
	assert.Contains(t, xml, "LJ.XMLRPC.getchallenge")
	assert.Contains(t, xml, "username")
	assert.Contains(t, xml, "testuser")
}

func TestDecodeValue_RawText(t *testing.T) {
	// XML-RPC allows <value>text</value> without type wrapper
	v := xmlrpcValue{RawText: "  hello world  "}
	result := decodeValue(v)
	assert.Equal(t, "hello world", result)
}

func TestDecodeValue_RawTextEmpty(t *testing.T) {
	// Empty RawText should return nil
	v := xmlrpcValue{RawText: ""}
	result := decodeValue(v)
	assert.Nil(t, result)
}

func TestDecodeValue_RawTextWhitespace(t *testing.T) {
	// Whitespace-only RawText should return nil after trimming
	v := xmlrpcValue{RawText: "   "}
	result := decodeValue(v)
	assert.Equal(t, "", result)
}
