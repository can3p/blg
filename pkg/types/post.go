package types

type PostHeaders map[string]any

type PostBody interface {
	// local -> remote
	ReplaceImages(map[string]string) error
	// local .md filename -> remote URL
	ReplaceLinks(map[string]string) error
	// Extract local .md file links
	ExtractLinks() ([]string, error)
	MaybeString() (string, error)
}

type Post struct {
	Headers PostHeaders
	Body    PostBody
}
