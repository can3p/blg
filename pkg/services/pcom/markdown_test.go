package pcom

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestExtractImages(t *testing.T) {
	var examples = []struct {
		body   string
		result []string
	}{
		{
			body: `this is a post

with a couple of images

![test1](test1.jpg)

![test2](test2.png)`,
			result: []string{"test1.jpg", "test2.png"},
		},
	}

	for idx, ex := range examples {
		p, err := parseBody(ex.body)

		assert.NoError(t, err, "example %d: parsing", idx+1)

		arr, err := p.ExtractImages()

		assert.NoError(t, err, "example %d: images extraction", idx+1)

		assert.Equal(t, ex.result, arr, "example %d: images extraction", idx+1)
	}
}

func TestReplaceImages(t *testing.T) {
	var examples = []struct {
		body   string
		m      map[string]string
		result string
	}{
		{
			body: `this is a post

with a couple of images

![test1](test1.jpg)

![test2](test2.png)`,
			m: map[string]string{
				"test1.jpg": "remote_id1",
			},
			result: `this is a post

with a couple of images

![test1](remote_id1)

![test2](test2.png)`,
		},
	}

	for idx, ex := range examples {
		p, err := parseBody(ex.body)

		assert.NoError(t, err, "example %d: parsing", idx+1)

		err = p.ReplaceImages(ex.m)

		assert.NoError(t, err, "example %d: images replacement", idx+1)

		res, err := p.MaybeString()

		assert.NoError(t, err, "example %d: body rendering", idx+1)

		assert.Equal(t, ex.result, res, "example %d:", idx+1)
	}
}
