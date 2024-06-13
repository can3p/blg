package livejournal

import "github.com/can3p/blg/pkg/types"

const endpoint = "/interface/xmlrpc"

type client struct {
	cfg types.Config
}

func (c *client) Push() error {
	return types.ErrNotImplemented
}

func createClient(cfg types.Config) types.Service {
	return &client{cfg}
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
