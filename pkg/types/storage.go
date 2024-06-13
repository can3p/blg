package types

const ConfigName = "posts.json"

type PostMeta struct {
	FileName string `json:"file_name"`
	RemoteID string `json:"remote_id"`
}

type ImageMeta struct {
	FileName string `json:"file_name"`
	RemoteID string `json:"remote_id"`
}

type StoredConfig struct {
	Login        string       `json:"login"`
	ServiceName  string       `json:"service_name"`
	CustomHost   string       `json:"custom_host,omitempty"`
	RemotePosts  []*PostMeta  `json:"remote_posts,omitempty"`
	RemoteImages []*ImageMeta `json:"remote_images,omitempty"`
}

type Config struct {
	Stored   StoredConfig
	Host     string
	Endpoint string
	Password string
}
