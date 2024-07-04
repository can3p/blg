package types

type RemotePost struct {
	ID        string `json:"id"`
	Hash      string `json:"hash"`
	Data      any    `json:"data"`
	UpdatedAt int64  `json:"updated_at"`
}

type RemotePosts struct {
	Version int64 `json:"version"`
	LastTS  int64 `json:"last_ts"`
	Posts   []*RemotePost
}
