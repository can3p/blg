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

func TestExtractLinks(t *testing.T) {
	var examples = []struct {
		body   string
		result []string
	}{
		{
			body:   `Just text, no links`,
			result: []string{},
		},
		{
			body:   `Check [my post](2024-01-01-test.md) for details`,
			result: []string{"2024-01-01-test.md"},
		},
		{
			body:   `[one](a.md) and [two](b.md) and [external](https://example.com)`,
			result: []string{"a.md", "b.md"},
		},
	}

	for idx, ex := range examples {
		p, err := parseBody(ex.body)

		assert.NoError(t, err, "example %d: parsing", idx+1)

		arr, err := p.ExtractLinks()

		assert.NoError(t, err, "example %d: links extraction", idx+1)

		assert.Equal(t, ex.result, arr, "example %d: links extraction", idx+1)
	}
}

func TestReplaceLinks(t *testing.T) {
	var examples = []struct {
		body   string
		m      map[string]string
		result string
	}{
		{
			body: `See [my post](2024-01-01-test.md) for details`,
			m: map[string]string{
				"2024-01-01-test.md": "https://user.livejournal.com/12345.html",
			},
			result: `See [my post](https://user.livejournal.com/12345.html) for details`,
		},
	}

	for idx, ex := range examples {
		p, err := parseBody(ex.body)

		assert.NoError(t, err, "example %d: parsing", idx+1)

		err = p.ReplaceLinks(ex.m)

		assert.NoError(t, err, "example %d: links replacement", idx+1)

		res, err := p.MaybeString()

		assert.NoError(t, err, "example %d: body rendering", idx+1)

		assert.Equal(t, ex.result, res, "example %d:", idx+1)
	}
}

func TestReplaceLinksWithResolver(t *testing.T) {
	// URL to filename mapping - simulates what we'd have after fetching posts
	urlToFilename := map[string]string{
		"https://pcom.com/posts/abc123": "that.md",
		"https://pcom.com/posts/def456": "this.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	var examples = []struct {
		name   string
		body   string
		result string
	}{
		{
			name:   "basic link resolution",
			body:   `check the post [this one](https://pcom.com/posts/def456)`,
			result: `check the post [this one](this.md)`,
		},
		{
			name:   "reverse link resolution",
			body:   `check the previous post [that one](https://pcom.com/posts/abc123)`,
			result: `check the previous post [that one](that.md)`,
		},
		{
			name:   "multiple links in same post",
			body:   `See [post A](https://pcom.com/posts/abc123) and [post B](https://pcom.com/posts/def456)`,
			result: `See [post A](that.md) and [post B](this.md)`,
		},
		{
			name:   "external link preserved",
			body:   `Check [external site](https://example.com/page)`,
			result: `Check [external site](https://example.com/page)`,
		},
		{
			name:   "mixed internal and external links",
			body:   `See [my post](https://pcom.com/posts/abc123) and [Google](https://google.com)`,
			result: `See [my post](that.md) and [Google](https://google.com)`,
		},
		{
			name:   "unknown internal link preserved",
			body:   `See [unknown post](https://pcom.com/posts/unknown999)`,
			result: `See [unknown post](https://pcom.com/posts/unknown999)`,
		},
		{
			name:   "link with anchor",
			body:   `See [section 1](https://pcom.com/posts/abc123#section1)`,
			result: `See [section 1](that.md#section1)`,
		},
		{
			name:   "local md link preserved",
			body:   `See [local](other.md)`,
			result: `See [local](other.md)`,
		},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			p, err := parseBody(ex.body)
			assert.NoError(t, err, "parsing")

			err = p.ReplaceLinksWithResolver(resolver)
			assert.NoError(t, err, "links replacement")

			res, err := p.MaybeString()
			assert.NoError(t, err, "body rendering")

			assert.Equal(t, ex.result, res)
		})
	}
}

func TestReplaceLinksWithResolver_NilResolver(t *testing.T) {
	body := `check [this](https://pcom.com/posts/abc123)`

	p, err := parseBody(body)
	assert.NoError(t, err)

	err = p.ReplaceLinksWithResolver(nil)
	assert.NoError(t, err)

	res, err := p.MaybeString()
	assert.NoError(t, err)

	// Links should be preserved as-is when resolver is nil
	assert.Equal(t, body, res)
}

func TestReplaceLinksWithResolver_CircularReferences(t *testing.T) {
	// Two posts that reference each other
	urlToFilename := map[string]string{
		"https://pcom.com/posts/post122": "that.md",
		"https://pcom.com/posts/post123": "this.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	// Post 122 (that.md) references post 123 (this.md)
	p1, _ := parseBody(`check the post [this one](https://pcom.com/posts/post123)`)
	_ = p1.ReplaceLinksWithResolver(resolver)
	res1, _ := p1.MaybeString()
	assert.Equal(t, `check the post [this one](this.md)`, res1)

	// Post 123 (this.md) references post 122 (that.md)
	p2, _ := parseBody(`check the previous post [that one](https://pcom.com/posts/post122)`)
	_ = p2.ReplaceLinksWithResolver(resolver)
	res2, _ := p2.MaybeString()
	assert.Equal(t, `check the previous post [that one](that.md)`, res2)
}

func TestReplaceLinksWithResolver_NestedLinks(t *testing.T) {
	urlToFilename := map[string]string{
		"https://pcom.com/posts/abc123": "post.md",
	}

	resolver := func(url string) (string, bool) {
		fname, ok := urlToFilename[url]
		return fname, ok
	}

	var examples = []struct {
		name   string
		body   string
		result string
	}{
		{
			name:   "link in bold",
			body:   `**[bold link](https://pcom.com/posts/abc123)**`,
			result: `**[bold link](post.md)**`,
		},
		{
			name:   "link in italic",
			body:   `*[italic link](https://pcom.com/posts/abc123)*`,
			result: `*[italic link](post.md)*`,
		},
		{
			name:   "link in list item",
			body:   `- See [this post](https://pcom.com/posts/abc123)`,
			result: `- See [this post](post.md)`,
		},
		{
			name:   "link in blockquote",
			body:   `> As mentioned in [this post](https://pcom.com/posts/abc123)`,
			result: `> As mentioned in [this post](post.md)`,
		},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			p, err := parseBody(ex.body)
			assert.NoError(t, err)

			err = p.ReplaceLinksWithResolver(resolver)
			assert.NoError(t, err)

			res, err := p.MaybeString()
			assert.NoError(t, err)

			assert.Equal(t, ex.result, res)
		})
	}
}
