package pcom

import "github.com/can3p/blg/pkg/types"

const endpoint = "/api"

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
			"pcom",
			"www.pcom.com",
			createClient))
}
