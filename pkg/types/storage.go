package types

type PostMeta struct {
	FileName string `json:"file_name"`
	RemoteID string `json:"remote_id"`
	Hash     string `json:"hash"`
}

type ImageMeta struct {
	FileName string `json:"file_name"`
	RemoteID string `json:"remote_id"`
	Hash     string `json:"hash"`
}

type StoredConfig struct {
	Version      int          `json:"version"`
	Login        string       `json:"login"`
	ServiceName  string       `json:"service_name"`
	CustomHost   string       `json:"custom_host,omitempty"`
	RemotePosts  []*PostMeta  `json:"remote_posts,omitempty"`
	RemoteImages []*ImageMeta `json:"remote_images,omitempty"`
}

type Config struct {
	Stored   StoredConfig
	RootPath string
	Host     string
}
