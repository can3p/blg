package types

import (
	"cmp"
)

type ServiceDefinition struct {
	Name        string
	DefaultHost string
	ServiceFunc ServiceFunc
}

func (sd ServiceDefinition) GetService(c *Config) (Service, error) {
	c.Host = cmp.Or(c.Stored.CustomHost, sd.DefaultHost)

	return sd.ServiceFunc(*c)
}

type Service interface {
	PreparePost(headers map[string]string, body string) (*Post, []string, error)
	UploadImage(string) (string, error)
	Create(p *Post) (string, error)
	Update(remoteID string, p *Post) error
	Delete(remoteID string) error
	PostURL(remoteID string) string
	NewPostTemplate(name string) string
	FetchPosts(updatedSince int64) ([]*RemotePost, []string, error)
	DownloadImage(string) ([]byte, error)
}

type ServiceFunc func(c Config) (Service, error)

func NewServiceDefinition(name, defaultHost string, sf ServiceFunc) ServiceDefinition {
	return ServiceDefinition{
		Name:        name,
		DefaultHost: defaultHost,
		ServiceFunc: sf,
	}
}

type ServiceRepo struct {
	defs map[string]ServiceDefinition
}

func (sr *ServiceRepo) Register(sd ServiceDefinition) {
	if sr.defs == nil {
		sr.defs = map[string]ServiceDefinition{}
	}

	sr.defs[sd.Name] = sd
}

func (sr *ServiceRepo) Get(name string) (ServiceDefinition, bool) {
	sd, ok := sr.defs[name]
	return sd, ok
}

func (sr *ServiceRepo) Services() []string {
	out := []string{}

	for k := range sr.defs {
		out = append(out, k)
	}

	return out
}

var DefaultServiceRepo = &ServiceRepo{}
