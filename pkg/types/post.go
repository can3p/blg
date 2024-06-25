package types

type PostHeaders map[string]any

type PostBody interface {
	// local -> remote
	ReplaceImages(map[string]string) error
	MaybeString() (string, error)
}

type Post struct {
	Headers PostHeaders
	Body    PostBody
}
