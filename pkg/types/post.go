package types

type PostState int

const (
	PostDraft PostState = iota
	PostPublished
)

type Post struct {
	Title    string
	Markdown string
	State    PostState
}
