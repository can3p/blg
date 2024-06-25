package livejournal

import (
	"github.com/can3p/blg/pkg/types"
)

const endpoint = "/interface/xmlrpc"

type client struct {
	cfg types.Config
}

func (c *client) Push() error {
	return types.ErrNotImplemented
}

func (c *client) PreparePost(fields map[string]string, body string) (*types.Post, []string, error) {
	return nil, nil, types.ErrNotImplemented
}

func (c *client) UploadImage(fname string) (string, error) {
	return "", types.ErrNotImplemented
}

// ```
// curl -v -H'Authorization: Bearer <api-key>' -XPOST -d'{ "subject": "test post", "md_body": "is saved\n\n![trololo](0190478c-5592-74ab-9d1a-5cdab598f2dd.png)", "visibility": "direct_only" }' http://localhost:8080/api/v1/posts
// {"data":{"id":"01904796-62f7-7a9a-a7bd-1595ed6d1663","public_url":"http://localhost:8080/posts/01904796-62f7-7a9a-a7bd-1595ed6d1663"}}%
// ```
func (c *client) Create(p *types.Post) (string, error) {
	return "", types.ErrNotImplemented
}

func (c *client) Update(remoteID string, p *types.Post) error {
	return types.ErrNotImplemented
}

func (c *client) Delete(remoteID string) error {
	return types.ErrNotImplemented
}

func init() {
	types.DefaultServiceRepo.Register(
		types.NewServiceDefinition(
			"pcom",
			"www.pcom.com",
			createClient))
}

func createClient(cfg types.Config) (types.Service, error) {
	return &client{cfg}, nil
}

func init() {
	types.DefaultServiceRepo.Register(
		types.NewServiceDefinition(
			"livejournal",
			"www.livejournal.com",
			createClient))

	types.DefaultServiceRepo.
		Register(types.NewServiceDefinition(
			"dreamwidth",
			"www.dreamwidth.org",
			createClient,
		))
}
